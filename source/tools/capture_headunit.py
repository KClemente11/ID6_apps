#!/usr/bin/env python3
"""Collect local compatibility evidence from an authorized BMW NBT Evo over SSH."""

import argparse
import hashlib
from ipaddress import IPv4Address
import json
from pathlib import Path
import re
import struct
import subprocess

FIRMWARE = re.compile(r"\bNBTevo_([A-Z][0-9]{5}[A-Z])\b", re.I)
DISPLAY = re.compile(rb"resolution:\s*x\s*=\s*([0-9]{2,4}),\s*y\s*=\s*([0-9]{2,4})\b", re.I)
FRAMEWORKS = ("libAppFramework.A.so", "libCoreFramework.A.so")
FRAMEWORK_DIR = "/opt/hmi/ID5/lib"
LEGACY = (
    "HostKeyAlgorithms=+ssh-rsa",
    "KexAlgorithms=+diffie-hellman-group14-sha1,diffie-hellman-group1-sha1",
    "Ciphers=+aes128-cbc,aes256-cbc,3des-cbc",
    "MACs=+hmac-sha1,hmac-sha1-96",
)


def enet_ip(value):
    try:
        address = IPv4Address(value)
    except ValueError as error:
        raise argparse.ArgumentTypeError(str(error)) from None
    if not address.is_link_local or address.packed[2] in (0, 255):
        raise argparse.ArgumentTypeError("use the head unit's wired 169.254.x.x ENET IP")
    return str(address)


def firmware_name(text):
    match = FIRMWARE.search(text)
    return "NBTevo_" + match.group(1).upper() if match else ""


def ssh_read(head, legacy, command, optional=False):
    args = ["ssh", "-T", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=5"]
    if legacy:
        for option in LEGACY:
            args.extend(("-o", option))
    result = subprocess.run([*args, "root@" + head, command], capture_output=True, timeout=60)
    if result.returncode:
        if optional:
            return b""
        detail = result.stderr.decode(errors="replace").strip()
        raise ValueError(f"{command} failed over SSH: {detail or result.returncode}")
    if len(result.stdout) > 4 << 20:
        raise ValueError(f"{command} returned more data than expected")
    return result.stdout


def capture(read, entered):
    detected = firmware_name(read("cat /opt/sys/etc/nbt_version.txt", optional=True).decode(errors="replace"))
    if entered and not FIRMWARE.fullmatch(entered):
        raise ValueError("enter the exact NBTevo firmware code shown on iDrive")
    entered = firmware_name(entered or "")
    if detected and entered and detected != entered:
        raise ValueError("the iDrive firmware code differs from the one reported over SSH")
    if not (detected or entered):
        raise ValueError("firmware code unavailable over SSH. Rerun with --firmware and the exact iDrive code")

    platform = read("uname -a").decode(errors="replace").split()
    if len(platform) < 4 or platform[0] != "QNX" or platform[-1] != "armle":
        raise ValueError("expected an ARM QNX head unit")
    display = DISPLAY.search(read("/opt/sys/bin/LayerManagerControl get screen 0", optional=True))
    width, height = (int(value) for value in display.groups()) if display else (0, 0)

    frameworks = {}
    for name in FRAMEWORKS:
        data = read("cat " + FRAMEWORK_DIR + "/" + name)
        if len(data) < 20 or data[:6] != b"\x7fELF\x01\x01" or struct.unpack_from("<HH", data, 16) != (3, 40):
            raise ValueError(f"{name} is not a little-endian ARM ELF32 shared object")
        frameworks[name] = data
    report = {
        "firmware": detected or entered,
        "firmware_source": "ssh" if detected else "manual",
        "os": "QNX",
        "os_release": platform[2],
        "architecture": "armle",
        "display": {"width": width, "height": height} if width and height else None,
        "frameworks": {name: {"sha256": hashlib.sha256(data).hexdigest(), "size_bytes": len(data)}
                       for name, data in frameworks.items()},
    }
    return report, frameworks


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--head", required=True, type=enet_ip, help="head unit SSH IP")
    parser.add_argument("--firmware", help="exact NBTevo code shown in iDrive if needed")
    parser.add_argument("--out", required=True, type=Path, help="new private output directory")
    parser.add_argument("--include-frameworks", action="store_true", help="also save both framework binaries for private ABI comparison")
    parser.add_argument("--legacy", action="store_true", help="allow legacy SSH algorithms for this ENET connection only")
    args = parser.parse_args()
    try:
        if args.out.exists():
            raise ValueError("output directory already exists")
        def read(command, optional=False):
            return ssh_read(args.head, args.legacy, command, optional)
        report, frameworks = capture(read, args.firmware)
        args.out.mkdir(mode=0o700)
        (args.out / "compatibility.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        if args.include_frameworks:
            for name, data in frameworks.items():
                (args.out / name).write_bytes(data)
        print(f"Saved local compatibility evidence to {args.out}")
    except (OSError, ValueError, subprocess.TimeoutExpired) as error:
        raise SystemExit(str(error)) from None


if __name__ == "__main__":
    main()
