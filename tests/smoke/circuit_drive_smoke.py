#!/usr/bin/env python3
"""Circuit drive smoke test.

Runs `circuit drive build-job --task "..."` against a live `pi --mode rpc`
process and asserts the loop reaches terminal.

Requirements: pi binary on PATH, model API key, go toolchain.
Run: python3 tests/smoke/circuit_drive_smoke.py
Exit: 0 on success, 1 on failure.
"""

import subprocess
import sys
import tempfile
import pathlib
import shutil
import os

REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]


def main() -> int:
    with tempfile.TemporaryDirectory() as workdir_str:
        workdir = pathlib.Path(workdir_str)
        machines_src = REPO_ROOT / "machines"
        machines_dst = workdir / "machines"
        machines_dst.mkdir(parents=True, exist_ok=True)
        shutil.copy(machines_src / "build-job.mch", machines_dst / "build-job.mch")
        shutil.copy(
            machines_src / "build-job.prompts.yaml",
            machines_dst / "build-job.prompts.yaml",
        )

        binary = workdir / "circuit"
        build = subprocess.run(
            [
                "go",
                "build",
                "-o",
                str(binary),
                "./cmd/circuit",
            ],
            cwd=REPO_ROOT,
            check=False,
        )
        if build.returncode != 0:
            print("FAIL: go build failed", file=sys.stderr)
            return 1

        env = os.environ.copy()
        env["PATH"] = f"{binary.parent}:{env.get('PATH', '')}"
        result = subprocess.run(
            [
                str(binary),
                "drive",
                "build-job",
                "--task",
                "Drive build-job idle -> running -> done.",
            ],
            cwd=workdir,
            capture_output=True,
            text=True,
            timeout=180,
            env=env,
        )
        print(result.stdout)
        if result.returncode != 0:
            print(result.stderr, file=sys.stderr)
            print("FAIL: drive command exited nonzero", file=sys.stderr)
            return 1

        for expected in (
            "advanced: idle -> running",
            "advanced: running -> done",
            "terminal: done",
        ):
            if expected not in result.stdout:
                print(f"FAIL: expected {expected!r} in drive output", file=sys.stderr)
                return 1

        print("PASS: circuit drive build-job reached done via real pi")
        return 0


if __name__ == "__main__":
    sys.exit(main())
