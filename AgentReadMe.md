# AgentReadMe

Use this runbook to install BoostDeck, ShiftDeck, and MHD Maps on an authorized BMW NBT Evo ID6 head unit, diagnose blank readings, or bring up a firmware version the installer does not recognize. Read [source/AGENTS.md](source/AGENTS.md) for the repository rules. Keep the vehicle parked for every live test.

## Choose the workflow

| Request | Where to start | QNX SDK needed |
| --- | --- | --- |
| Install on a listed firmware with matching frameworks | Existing Mac or Windows installer over ENET | No |
| Inspect firmware, display, or framework compatibility | Temporary SSH and the read-only head-unit capture below | No |
| Figure out why one gauge is blank | Wired ENET and the read-only signal probe | No |
| Adjust existing XML positions or regenerate artwork with the same model and asset IDs | Layout generators and an existing matching native payload | No for XML and artwork alone |
| Change a worker's diagnostic requests, decoder, or timing, or change native plugin behavior | Rebuild the ARM payload and run the tests | Yes |
| Package the desktop installer using a complete matching payload | Go and macOS packaging tools | No |

The vehicle already runs QNX. The build toolchain for native changes is QNX SDP 6.5 with the Linux x86 host tools and ARM GCC 4.4.2. Use the build host for cross-compiling and the computer physically connected to ENET for live work. [Build instructions](source/docs/BUILD.md)

## Connect and enable temporary SSH

1. Confirm the owner has authorized the work and the car is parked with ignition on and iDrive awake. Connect the computer directly to the OBD port with a wired ENET cable. Use a USB Ethernet adapter if necessary.
2. Record the exact `NBTevo_...` navigation software version from iDrive. Identify the computer's local ENET IP, the car's diagnostic gateway IP, and the head unit's SSH IP. They are distinct. On macOS use `ifconfig`, and on Windows use `ipconfig`. The gauge installer's **Find car** can discover the gateway and, where supported, the head unit. Use the SSH IP shown by the unlock tool if needed.
3. If SSH is closed and the tool supports the connected head unit, use the official **Bavarian SSH Client** for the operator's OS. Compare the download hash against the publisher's checksum. Open the client on the ENET-connected computer, let it identify the car, verify it has selected the right head unit, and choose **Enable SSH**. Use the owner's authorized UI session to automate the same clicks if available. This project has no documented Bavarian CLI or API. If Bavarian cannot identify or unlock the exact build, record the error and use an owner-approved method documented for that firmware. An installer firmware mismatch is a separate issue from SSH unlock support. [Bavarian SSH Client downloads](https://ssh.bavarian-aa.uk/)
4. Confirm TCP port 22 responds on the head unit. For example, use `nc -vz -G 3 HEAD_IP 22` on macOS or `Test-NetConnection HEAD_IP -Port 22` in Windows PowerShell. Confirm the SSH host key belongs to that head unit before trusting it. If it changes, stop and re-identify the endpoint. Enter the authorized SSH password locally when prompted. Keep it out of logs, commands, and commits. Bavarian's temporary SSH closes on a head-unit reboot, so enable it again if a later recovery session needs it. [Bavarian SSH Client](https://ssh.bavarian-aa.uk/)

Replace `HEAD_IP` and other placeholders with the values from the connected car. Keep the connection local to ENET. If the agent has no access to that computer, have the operator connect, unlock, and return sanitized results.

## Install a matching build

1. Open the installer for the computer's OS. Run **Check compatibility**, verify the presented SSH fingerprint, then choose **Trust this fingerprint and check**. Select the actual iDrive output resolution if automatic detection fails. The panel's diagonal and its advertised resolution are insufficient to choose the layout.
2. Proceed with **Install all three apps** only after compatibility passes. The installer reads the firmware, checks the live framework pair, verifies gateway reachability, stages complete app folders, and reads back the installed files. Leave iDrive on and ENET connected. If it stops, preserve the full status, live folders, and staging or backup path. Use **Clear lock and install** only when the installer offers it and the operator has confirmed every other installer is closed. An existing lock with incomplete or ambiguous publication needs inspection.
3. When the installer reports **Installed and verified all three apps**, have the operator hold the volume/power knob while parked until the screen goes dark, usually around 30 seconds. Release, wait for iDrive to load, and verify the menus under **My Vehicle**. A successful file install does not prove gauge accuracy or confirm an actual map change. Check those separately while parked. [Install and recovery guide](source/README.md#connect-and-install)

### Local automation interface

From `source/` with the payload staged, run `go run . -no-browser` on the ENET-connected machine. Capture the local URL printed to stdout. Keep its random path private. Send JSON POSTs relative to that full URL, with `Content-Type: application/json` and `Origin` equal to the URL's scheme, host, and port. The interface listens on loopback. [Implementation](source/main.go)

- `discover` accepts `{}` and returns candidates containing `Head`, `Gateway`, and `Local`. Use the operator-confirmed candidate.
- `inspect` accepts those fields plus `Version`, `DisplayResolution`, `Fingerprint`, and `Password`. Use an empty `DisplayResolution` for auto or a value such as `1280x480`. On the first request, leave `Fingerprint` empty. Verify the returned host key and repeat with the verified fingerprint and the locally entered password. Wait for `Ready: true`.
- `install` rechecks the vehicle and uses the same fields. Preserve the returned status and backup path. Success is the explicit verification message. `Ready` becomes false after an install attempt.
- `recover` uses the same fields. Use it only when the preceding install returned `CanRecover: true` and the operator confirmed no other installer is active. The server rechecks the recovery conditions before changing the lock.

## Bring up an unlisted firmware

SSH access to a build does not establish app compatibility. Keep the installer gate closed until the port has evidence.

1. Collect firmware, OS/architecture, display resolution when available, and both live framework hashes using the read-only capture tool. The desktop capture uses OpenSSH and a previously verified known-hosts entry. For first contact, use `ssh root@HEAD_IP uname -a` from the ENET laptop and accept the key only after verifying it. If legacy algorithms are required, scope them to this ENET endpoint in the SSH client configuration, or use the capture tool’s `--legacy` flag after the key is trusted. The capture may prompt more than once when the unit uses password authentication. Run from `source/` and replace the placeholders.

   ```sh
   python3 tools/capture_headunit.py --head HEAD_IP --firmware NBTevo_EXACT_CODE --out /private/nbt-candidate
   ```

   Omit `--firmware` if the unit reports it over SSH. Add `--include-frameworks` only when doing private ABI comparison. Keep these two library files and any private reference libraries in local directories. Share hashes and sanitized compatibility results with contributors.
2. Compare the pair to `profiles` in `source/install.go`. If both hashes match an existing profile, record the match as evidence. If either differs, run `source/native/tools/audit_framework_abi.py` against the known-good reference and candidate pairs for each plugin. Check the imported symbols, constructors and virtual-table relocation slots. Rebuild against private matching framework inputs if needed. If the ABI differs, resolve the mismatch before attempting a parked load. Static checks alone do not prove that the plugin will load.
3. Create a local candidate profile for the exact firmware and hashes. Build and run the normal installer checks on a parked, authorized test car. Verify load after reboot, menu geometry, controller focus, option actions, visibility-based worker startup and shutdown, and clean rollback. If touch is present, verify it separately. Record the scope of the test before adding public support. [ABI and build commands](source/docs/BUILD.md)
4. Check the DME and EGS separately. A head-unit match says nothing about whether the vehicle supports a particular coolant or oil pressure request. Follow the signal workflow below.

## Debug blank gauges or port a new engine

Exit the gauge pages and other diagnostic clients. Use the read-only ENET probe from the laptop. Start with `010c` for an RPM control, `0105` for a standard coolant comparison, and `224300` and `22586f` for the current BMW coolant and oil-pressure candidates. Use ECU `12` for DME and `18` for EGS. Record request and reply, ECU identity, length, sign, endian order, scaling, units, invalid values, and the independent measurement under the same conditions. Keep unverified readings blank. Add an ECU-specific profile when a new variant needs a different decoder. [Probe commands and interpretation](source/docs/PORTING.md) · [Current signal table](source/docs/SIGNALS.md)

Run `go test -race ./...`, `make -C native test`, and `python3 -m unittest discover -s tools -p 'test_*.py'` from `source/`. State separately what passed on the host, in emulation, and on a car. Publish only reviewed source and sanitized evidence. Leave VINs, credentials, framework files, private captures, and full staging backups out of the repo.
