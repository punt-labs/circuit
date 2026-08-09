#!/usr/bin/env python3
"""Pi RPC smoke test for circuit extension.

Requires: pi binary, model API key, go toolchain.
Run: python3 tests/smoke/pi_rpc_smoke.py
Exit: 0 on success, 1 on failure.
"""

import json
import subprocess
import sys
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
    stderr=subprocess.PIPE,
    text=True,
    cwd=".",
)

assert proc.stdin is not None
assert proc.stdout is not None

failures = 0

try:
    for request_id, message, expected in COMMANDS:
        proc.stdin.write(
            json.dumps({"id": request_id, "type": "prompt", "message": message})
            + "\n"
        )
        proc.stdin.flush()

        started = time.time()
        matched = False
        while time.time() - started < 30:
            line = proc.stdout.readline()
            if not line:
                break
            event = json.loads(line)
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
    proc.terminate()

sys.exit(1 if failures > 0 else 0)
