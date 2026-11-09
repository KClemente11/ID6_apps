# NBT Evo ID6 apps
<img width="1200" height="900" alt="08_mhd_maps" src="https://github.com/user-attachments/assets/99f5af53-aa83-4571-a89c-18e2b66755e1" />
<img width="1200" height="900" alt="07_boostdeck_ambient" src="https://github.com/user-attachments/assets/ce269d0a-6aec-44ac-a933-418afd8aca7d" />
<img width="1200" height="900" alt="06_boostdeck_night" src="https://github.com/user-attachments/assets/82ece074-bae8-4c02-b38f-066776e58220" />
<img width="1200" height="900" alt="05_shiftdeck" src="https://github.com/user-attachments/assets/e8fc4394-917a-47cd-9315-531572c12ccf" />
<img width="1200" height="900" alt="04_hud_orange" src="https://github.com/user-attachments/assets/00706e6f-955a-41bb-8655-3524c2e7f03d" />
<img width="1200" height="900" alt="03_hud_white" src="https://github.com/user-attachments/assets/3f90481c-6306-4bf6-a6fc-091444d343b0" />
<img width="1200" height="900" alt="02_boostdeck_closeup" src="https://github.com/user-attachments/assets/4690994c-b49e-4b11-be56-4836ec73eec5" />
<img width="1200" height="900" alt="01_boostdeck_daylight" src="https://github.com/user-attachments/assets/f653b18a-df0d-4046-84a8-d96e3f3ea245" />



Park your car, connect it to your computer with ENET, and hand this repository to an AI agent that can run commands on that computer. The agent uses [AgentReadMe.md](AgentReadMe.md) to identify your head unit, check compatibility, and install BoostDeck, ShiftDeck, and MHD Maps under **My Vehicle** when the checks pass.

## Start with your car and an agent

1. Park the car, switch on the ignition, and wait for iDrive to start. Connect the car's OBD port to the computer with a wired ENET cable. Use a USB Ethernet adapter if needed.
2. Open an AI agent with terminal access on that computer, such as Codex running locally. Give it the prompt below. The agent needs access to the computer plugged into the car.
3. Stay with the car. The agent may need you to enable temporary SSH with the unlock tool, read the firmware version from iDrive, or verify the SSH fingerprint. Keep iDrive awake and ENET connected through the install. Restart iDrive only after the agent confirms the installed files were verified, then check the menus while parked.

### Copy this into your agent

> My car is parked, iDrive is awake, and this computer is connected to the car over wired ENET. Read https://github.com/inotbarnacle/ID6_apps_prerelease/blob/main/AgentReadMe.md and follow it on this computer to install BoostDeck, ShiftDeck, and MHD Maps. Identify my head unit and check its firmware, framework hashes, and active display resolution. Use the local installer automation when the checks pass. If the firmware is unlisted or the frameworks differ, start with the read-only compatibility capture and follow the porting workflow. Tell me when you need me at the car. Have me enter any SSH password locally. After installation, tell me when to restart iDrive and what to verify.

## What the agent checks

The published installer has profiles for `NBTevo_N18092I`, `NBTevo_V20153I`, and `NBTevo_Y24073J`. The agent checks the exact firmware and both framework hashes before writing files. An unlisted build needs compatibility work and parked validation before it can be treated as supported. [Unlisted firmware workflow](AgentReadMe.md#bring-up-an-unlisted-firmware)

BoostDeck's BMW-specific gauge definitions come from a 2016 340i and may vary by ECU. ShiftDeck uses standard OBD RPM and speed. MHD Maps requires a compatible MHD+ multi-map tune. Check the active slot in MHD separately if you test map switching. [Signal and map details](source/README.md)


