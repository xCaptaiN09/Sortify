#!/usr/bin/env python3
"""Cross-compile Sortify helpers without installing anything on the phone."""
from __future__ import annotations

import argparse
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
TARGETS = {'arm64': ('android', 'arm64'), 'arm': ('linux', 'arm'),
           'x64': ('linux', 'amd64'), 'x86': ('linux', '386')}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--go', default=shutil.which('go'))
    parser.add_argument('--host-only', action='store_true')
    arguments = parser.parse_args()
    if not arguments.go:
        parser.error('Go 1.24 or newer is required; use --go /path/to/go')
    destination = ROOT / 'build'
    destination.mkdir(exist_ok=True)
    targets = {'host': (sys.platform, '')} if arguments.host_only else TARGETS
    for name, (operating_system, architecture) in targets.items():
        target = destination / 'bin' / name / 'sortify'
        target.parent.mkdir(parents=True, exist_ok=True)
        environment = {**os.environ, 'CGO_ENABLED': '0'}
        if not arguments.host_only:
            environment.update({'GOOS': operating_system, 'GOARCH': architecture})
            if architecture == 'arm':
                environment['GOARM'] = '7'
        command = [arguments.go, 'build', '-trimpath', '-buildvcs=false', '-ldflags=-s -w', '-o', str(target), './cmd/sortify']
        subprocess.run(command, cwd=ROOT / 'core', env=environment, check=True)
        print(f'Built {name}: {target}', flush=True)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
