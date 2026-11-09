# NBT Evo ID6 apps

## Gallery

<table>
  <tr>
    <td><img src="images/01_boostdeck_daylight.jpg" width="300"></td>
    <td><img src="images/02_boostdeck_closeup.jpg" width="300"></td>
  </tr>
  <tr>
    <td><img src="images/05_shiftdeck.jpg" width="300"></td>
    <td><img src="images/06_boostdeck_night.jpg" width="300"></td>
  </tr>
  <tr>
    <td><img src="images/07_boostdeck_ambient.jpg" width="300"></td>
    <td><img src="images/08_mhd_maps.jpg" width="300"></td>
  </tr>
</table>

## Coming Soon (In Development)

<table>
  <tr>
    <td><img src="images/03_hud_white.jpg" width="300"></td>
    <td><img src="images/04_hud_orange.jpg" width="300"></td>
  </tr>
</table>




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


