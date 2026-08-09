#!/usr/bin/env python3
"""Pi RPC smoke test for circuit extension.

Requires: pi binary, model API key, go toolchain.
Run: python3 tests/smoke/pi_rpc_smoke.py
Exit: 0 on success, 1 on failure.
"""

import json
import os
import subprocess
import sys
import threading
import time

COMMANDS = [
    ("list", "/circuit list", "build-job"),
    ("start", "/circuit start build-job", "started: build-job"),
    ("status1", "/circuit status", "current: idle"),
    ("advance1", "/circuit advance start", "advanced: idle -> running"),
    ("status2", "/circuit status", "current: running"),
    ("advance2", "/circuit advance finish", "advanced: running -> done"),
]

proc = subprocess.Popen(
    ["pi", "--mode", "rpc", "--no-session", "--approve"],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.DEVNULL,
    text=True,
    cwd=".",
)

assert proc.stdin is not None
assert proc.stdout is not None

# Set stdout to non-blocking via a reader thread so readline cannot hang forever.
lines: list[str] = []
stop_reader = threading.Event()


def reader() -> None:
    assert proc.stdout is not None
    while not stop_reader.is_set():
        line = proc.stdout.readline()
        if not line:
            break
        lines.append(line)


reader_thread = threading.Thread(target=reader, daemon=True)
reader_thread.start()

failures = 0

try:
    for request_id, message, expected in COMMANDS:
        proc.stdin.write(
            json.dumps({"id": request_id, "type": "prompt", "message": message})
            + "\n"
        )
        proc.stdin.flush()

        deadline = time.time() + 30
        matched = False
        while time.time() < deadline:
            if not lines:
                time.sleep(0.1)
                continue
            raw = lines.pop(0)
            event = json.loads(raw)
            if (
                event.get("type") != "extension_ui_request"
                or event.get("method") != "notify"
            ):
                continue
            actual = event.get("message", "")
            matched = expected in actual
            if matched:
                print(f"PASS {request_id}: {expected}")
            else:
                print(f"FAIL {request_id}: expected {expected!r} in {actual!r}")
                failures += 1
            break

        if not matched and failures == 0:
            print(f"FAIL {request_id}: timed out waiting for response")
            failures += 1
finally:
    stop_reader.set()
    proc.terminate()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()
        proc.wait()

sys.exit(1 if failures > 0 else 0)
