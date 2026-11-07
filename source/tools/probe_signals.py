#!/usr/bin/env python3
"""Read explicit BMW diagnostic signals over wired HSFZ while parked."""

import argparse
from ipaddress import IPv4Address
import json
import re
import socket
import struct
import time


def enet_address(value):
    address = IPv4Address(value)
    if not address.is_link_local or address.packed[2] in (0, 255):
        raise argparse.ArgumentTypeError("use a usable 169.254.x.x ENET address")
    return str(address)


def read_request(value):
    if not re.fullmatch(r"01[0-9a-fA-F]{2}|22[0-9a-fA-F]{4}", value):
        raise argparse.ArgumentTypeError("use a read-only OBD PID (0105) or UDS DID (224300)")
    return bytes.fromhex(value)


def exact(connection, length, deadline):
    data = bytearray()
    while len(data) < length:
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError("no matching ECU response")
        connection.settimeout(remaining)
        part = connection.recv(length - len(data))
        if not part:
            raise ConnectionError("gateway closed the connection")
        data.extend(part)
    return data


def exchange(connection, ecu, request):
    connection.settimeout(2)
    connection.sendall(struct.pack(">IHBB", len(request) + 2, 1, 0xF4, ecu) + request)
    deadline = time.monotonic() + 2
    prefix = bytes((request[0] + 0x40,)) + request[1:]
    while True:
        length, control = struct.unpack(">IH", exact(connection, 6, deadline))
        if length > 8192:
            raise ValueError("oversized HSFZ frame")
        payload = exact(connection, length, deadline)
        if control == 0x12:
            connection.sendall(struct.pack(">IHBB", 2, 0x13, 0, 0xF4))
        if control != 1 or payload[:2] != bytes((ecu, 0xF4)):
            continue
        reply = bytes(payload[2:])
        if reply.startswith(prefix):
            return {"reply": reply.hex(" "), "data": reply[len(prefix):].hex(" ")}
        if len(reply) == 3 and reply[:2] == bytes((0x7F, request[0])) and reply[2] != 0x78:
            return {"reply": reply.hex(" "), "nrc": f"{reply[2]:02X}"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--local", required=True, type=enet_address, help="computer's wired ENET IPv4")
    parser.add_argument("--gateway", required=True, type=enet_address, help="diagnostic gateway IPv4")
    parser.add_argument("--ecu", choices=("12", "18"), default="12", help="12 for DME or 18 for EGS")
    parser.add_argument("requests", nargs="+", type=read_request)
    args = parser.parse_args()
    if args.local == args.gateway or len(args.requests) > 12:
        parser.error("use separate local and gateway IPs and at most 12 explicit requests")
    try:
        for request in args.requests:
            result = {"ecu": args.ecu, "request": request.hex(" ")}
            try:
                # A separate connection prevents a late response from matching a later read.
                with socket.create_connection((args.gateway, 6801), timeout=2, source_address=(args.local, 0)) as connection:
                    result.update(exchange(connection, int(args.ecu, 16), request))
            except TimeoutError as error:
                result["error"] = str(error)
            print(json.dumps(result), flush=True)
            time.sleep(0.2)
    except (OSError, ValueError) as error:
        raise SystemExit(str(error)) from None


if __name__ == "__main__":
    main()
