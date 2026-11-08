import argparse
import struct
import unittest
from unittest.mock import patch

from capture_headunit import FRAMEWORKS, capture, enet_ip, ssh_read


def mock_head(firmware=b"NBTevo_A20000I", valid_frameworks=True):
    library = b"\x7fELF\x01\x01" + bytes(10) + struct.pack("<HH", 3, 40) + b"example"
    def read(command, optional=False):
        if command == "cat /opt/sys/etc/nbt_version.txt":
            return firmware
        if command == "uname -a":
            return b"QNX private-hostname 6.5.0 build board armle"
        if command.endswith("get screen 0"):
            return b"- resolution: x=1280, y=480\n"
        if command in ("cat /opt/hmi/ID5/lib/" + name for name in FRAMEWORKS):
            return library if valid_frameworks else b"invalid"
        raise AssertionError(command)
    return read


class CaptureTest(unittest.TestCase):
    def test_accepts_an_unlisted_firmware_and_omits_hostname(self):
        report, files = capture(mock_head(), None)
        self.assertEqual(report["firmware"], "NBTevo_A20000I")
        self.assertEqual(report["firmware_source"], "ssh")
        self.assertEqual(report["display"], {"width": 1280, "height": 480})
        self.assertNotIn("private-hostname", str(report))
        self.assertEqual(set(files), set(FRAMEWORKS))
        self.assertTrue(all(len(f["sha256"]) == 64 for f in report["frameworks"].values()))

    def test_manual_version_and_invalid_framework(self):
        report, _ = capture(mock_head(b""), "NBTevo_A20000I")
        self.assertEqual(report["firmware_source"], "manual")
        with self.assertRaisesRegex(ValueError, "differs"):
            capture(mock_head(), "NBTevo_A20001I")
        with self.assertRaisesRegex(ValueError, "ELF32"):
            capture(mock_head(valid_frameworks=False), None)
        with self.assertRaises(argparse.ArgumentTypeError):
            enet_ip("192.168.1.2")

    @patch("capture_headunit.subprocess.run")
    def test_keeps_strict_host_key_checking_in_legacy_mode(self, run):
        run.return_value.returncode = 0
        run.return_value.stdout = b"example"
        self.assertEqual(ssh_read("169.254.10.3", True, "uname -a"), b"example")
        args = run.call_args.args[0]
        self.assertIn("StrictHostKeyChecking=yes", args)
        self.assertIn("HostKeyAlgorithms=+ssh-rsa", args)
        self.assertEqual(args[-2:], ["root@169.254.10.3", "uname -a"])


if __name__ == "__main__":
    unittest.main()
