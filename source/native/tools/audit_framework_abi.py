"""Compare supplied BMW ELF layouts without executing them (pyelftools + capstone).

Instruction-prefix comparisons stop before the first in-function PC-relative
literal pool. They do not compare the pool's relocated addresses or its strings.
Reports contain hashes and symbol identities, never library payload bytes.
"""
import argparse
import hashlib
import json
import struct
from pathlib import Path

from capstone import Cs, CS_ARCH_ARM, CS_MODE_ARM
from capstone.arm import ARM_OP_MEM, ARM_REG_PC
from elftools.elf.elffile import ELFFile


def inventory(path):
    with path.open('rb') as f:
        elf = ELFFile(f)
        assert elf.elfclass == 32 and elf.little_endian and elf['e_machine'] == 'EM_ARM'
        table = elf.get_section_by_name('.dynsym')
        symbols, relocations = {}, {}
        for s in table.iter_symbols():
            if not s['st_size'] or not isinstance(s['st_shndx'], int):
                continue
            sec = elf.get_section(s['st_shndx'])
            offset = s['st_value'] - sec['sh_addr']
            data = sec.data()[offset:offset + s['st_size']]
            symbols[s.name] = {'address': s['st_value'], 'data': data,
                               'type': s['st_info']['type']}
        for sec in elf.iter_sections():
            if sec['sh_type'] not in ('SHT_REL', 'SHT_RELA'):
                continue
            linked = elf.get_section(sec['sh_link'])
            for r in sec.iter_relocations():
                relocations[r['r_offset']] = [r['r_info_type'], linked.get_symbol(r['r_info_sym']).name]
        comment = elf.get_section_by_name('.comment')
        return symbols, relocations, comment.data().decode('ascii', 'replace').strip('\0') if comment else None


def prefix(symbol):
    data, address = symbol['data'], symbol['address']
    stop = len(data)
    decoder = Cs(CS_ARCH_ARM, CS_MODE_ARM)
    decoder.detail = True
    for insn in decoder.disasm(data, address):
        if insn.address - address >= stop:
            break
        for operand in insn.operands:
            if operand.type == ARM_OP_MEM and operand.mem.base == ARM_REG_PC:
                offset = insn.address + 8 + operand.mem.disp - address
                if 0 <= offset < stop:
                    stop = offset
    return data[:stop]


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--reference-dir', type=Path, required=True)
    p.add_argument('--installed-dir', type=Path, required=True)
    p.add_argument('--plugin', type=Path, required=True)
    p.add_argument('--report', type=Path, required=True)
    a = p.parse_args()
    with a.plugin.open('rb') as f:
        imports = {s.name for s in ELFFile(f).get_section_by_name('.dynsym').iter_symbols()
                   if s['st_shndx'] == 'SHN_UNDEF' and 'bmw' in s.name}
    report = {'method': 'Offline ELF bytes, relocation tables and Capstone ARM instruction-prefix boundaries',
              'native_execution': False, 'frameworks': [], 'failures': [],
              'limitations': ['Literal pools, runtime values and transitive callees are not semantically compared.',
                              'This comparison does not prove the reconstructed reference SDK correct.',
                              'Compiler behavior, HMI integration and lifecycle still require parked target tests.']}
    for name in ('libAppFramework.A.so', 'libCoreFramework.A.so'):
        rp, ip = a.reference_dir / name, a.installed_dir / name
        ref, rr, rc = inventory(rp)
        car, cr, cc = inventory(ip)
        entry = {'name': name, 'reference_sha256': hashlib.sha256(rp.read_bytes()).hexdigest(),
                 'installed_sha256': hashlib.sha256(ip.read_bytes()).hexdigest(),
                 'installed_compiler_comment': cc, 'reference_compiler_comment': rc,
                 'function_comparisons': [], 'vtables': []}
        selected = imports | {n for n in car if 'PluginAppC1E' in n or 'PluginAppC2E' in n}
        for n in sorted(selected):
            if n not in ref or n not in car or car[n]['type'] != 'STT_FUNC':
                continue
            rb, cb = prefix(ref[n]), prefix(car[n])
            same = rb == cb and len(ref[n]['data']) == len(car[n]['data'])
            entry['function_comparisons'].append({'symbol': n, 'installed_address': hex(car[n]['address']),
                'function_bytes': len(car[n]['data']), 'instruction_prefix_bytes': len(cb),
                'instruction_prefix_sha256': hashlib.sha256(cb).hexdigest(), 'same_size_and_prefix': same})
            if not same:
                report['failures'].append(n + ': size/instruction prefix differs')
        for n in ('_ZTVN3bmw9PluginAppE', '_ZTVN3bmw8RunnableE'):
            if n not in ref or n not in car:
                continue
            rs, cs = ref[n], car[n]
            slots = []
            for offset in range(0, len(rs['data']), 4):
                left, right = rr.get(rs['address'] + offset), cr.get(cs['address'] + offset)
                slots.append({'byte_offset': offset, 'relocation': right,
                              'equal': left == right and rs['data'][offset:offset+4] == cs['data'][offset:offset+4]})
            same = len(rs['data']) == len(cs['data']) and all(s['equal'] for s in slots)
            entry['vtables'].append({'symbol': n, 'same_bytes_and_relocation_slots': same, 'slots': slots})
            if not same:
                report['failures'].append(n + ': vtable differs')
        report['frameworks'].append(entry)
    comparisons = sum(len(x['function_comparisons']) for x in report['frameworks'])
    assert comparisons >= 35, 'Incomplete framework function coverage'
    assert any(v['symbol'] == '_ZTVN3bmw9PluginAppE'
               for x in report['frameworks'] for v in x['vtables']), 'Missing PluginApp vtable'
    report['compared_functions'] = comparisons
    report['static_comparison_passed'] = not report['failures']
    a.report.write_text(json.dumps(report, indent=2) + '\n')
    assert not report['failures'], report['failures']
    print('Framework comparison passed:', comparisons, 'function prefixes and PluginApp vtable; no target execution.')


if __name__ == '__main__':
    main()
