# ID6_apps_prerelease
# NBT Evo Gauge Installer

Parked preview for `NBTevo_N18092I`, `NBTevo_V20153I`, and `NBTevo_Y24073J`. The installer puts **BoostDeck**, **ShiftDeck**, and **MHD Maps** under My Vehicle. It checks both on-car HMI framework fingerprints before uploading. The preview has passed static framework checks and desktop installer tests. It still needs an on-car test on each of these builds.

BoostDeck shows boost, AFR, timing, intake air, transmission, rail pressure, oil pressure, and coolant. Its BMW-specific signal definitions come from the reference 2016 340i. Some cars will return fewer signals or use different definitions. Compare the values with a known-good logger before relying on them. ShiftDeck reads standard OBD RPM and speed and starts with a 6,500 RPM shift-light target.

## Connect and install

1. Park the car, switch on the ignition, and plug a wired ENET cable into the OBD port and your computer. Use a USB Ethernet adapter if needed.
2. Enable temporary SSH with an unlock tool that supports the exact head unit. [Bavarian SSH Client downloads](https://ssh.bavarian-aa.uk/)
3. Open **NBT Evo Gauge Installer**. The Mac ZIP contains a universal app for Apple silicon and Intel. The Windows ZIP contains an x64 executable. The app opens its interface in your browser on localhost. The Mac build is ad-hoc signed and the Windows build is unsigned.
4. Click **Find car**. The gateway IP and head-unit SSH IP are separate. If discovery only fills the gateway, take the SSH IP from the unlock tool. For manual troubleshooting, `ifconfig` on macOS or `ipconfig` on Windows shows the computer's ENET adapter. On macOS, `arp -an` and on Windows, `arp -a` show ENET neighbors the computer has seen.
5. Click **Check compatibility**. The first successful SSH handshake fills in the head unit's host key fingerprint. Confirm the SSH IP belongs to your connected car and compare the fingerprint with a trusted copy if you have one. Then click **Trust this fingerprint and check**. The installer expects the same host key for later SSH connections in this session. If iDrive does not expose its firmware string over SSH, enter the full NBTevo code from the navigation version information screen. The installer checks the live framework fingerprints even when you supply the code.
6. Confirm that the installer shows the expected firmware and display size, then click **Install all three apps**. Restart iDrive while parked and open BoostDeck, ShiftDeck, or MHD Maps under **My Vehicle**. Check the layouts, controller navigation, and taps on native buttons if the car has touch. Gauge data and map requests use the vehicle's diagnostic connection.

## MHD Maps

On a car already running a compatible [MHD+ multi-map tune](https://mhd-tuning.atlassian.net/wiki/spaces/MFS2/pages/11763743/MHD%2BSuite), open **My Vehicle → MHD Maps**. The four tiles send a map request with one press. For the parked test, select a slot you have configured for the fuel in the car. The confirmation text reflects the DME's acknowledgement of the request. Verify the active slot separately in MHD. A timeout leaves the outcome unknown, so check the active slot before another press.

## Display and touch

The original compatibility reports identify the firmware and HMI libraries. The installer reads the active central display size from the live head unit. It supports layout profiles for 800 × 480, 1280 × 480, and 1440 × 540. An unrecognized or unavailable display size stops installation. The 1440 × 540 profile scales the known layout proportionally. The 800 × 480 profile fits the bars into narrower columns and preserves the aspect ratio of the shift graphics and map tiles. These profiles need a visual check on the car.

Touch coding, when available in the factory CarPlay configuration, appears in the compatibility result. The gauge screens use native iDrive controls. The controller and OPTIONS button provide access to settings, and touch-equipped cars can test tapping the settings buttons. The installer leaves touch coding alone.

## Recovery

The installer stages and reads back every file before publishing. If it replaces an existing BoostDeck, ShiftDeck, or MHD Maps directory, it retains the old one in the backup directory displayed by the installer. Publication moves only these three app folders. Save the backup path before restarting iDrive.

If an app misbehaves after restart, enable temporary SSH. Move the installed app directory out of `/var/opt/hmi/data/ID5/Plugins` into a separate recovery directory. Restore its `old-BoostDeck`, `old-ShiftDeck`, or `old-MhdMaps` backup from the reported location if there was a prior install. Then restart iDrive while parked.
