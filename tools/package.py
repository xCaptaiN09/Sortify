#!/usr/bin/env python3
"""Assemble an offline manager-installable Sortify ZIP from built native helpers."""
from __future__ import annotations

import argparse
import hashlib
from pathlib import Path
import sys
import zipfile

ROOT = Path(__file__).resolve().parents[1]
ARCHITECTURES = ('arm64', 'arm', 'x64', 'x86')
MODULE_FILES = ('module.prop', 'customize.sh', 'action.sh', 'service.sh',
                'uninstall.sh', 'sortify.sh', 'LICENSE', 'README.md', 'CHANGELOG.md', 'banner.png')


def assemble(binary_dir: Path, output: Path) -> None:
    if output.name in ('SHA256SUMS', 'SHA256SUMS.tmp'):
        raise ValueError(f'ZIP output uses a reserved checksum filename: {output}')
    files: dict[str, bytes] = {}
    for architecture in ARCHITECTURES:
        path = binary_dir / architecture / 'sortify'
        if not path.is_file():
            raise ValueError(f'Missing native helper for {architecture!r}: {path}')
        payload = path.read_bytes()
        if not payload.startswith(b'\x7fELF'):
            raise ValueError(f'Native helper is not an ELF executable: {path}')
        files[f'bin/{architecture}/sortify'] = payload
    for name in MODULE_FILES:
        path = ROOT / name
        if not path.is_file():
            raise ValueError(f'Missing module file: {path}')
        files[name] = path.read_bytes()
    for path in sorted((ROOT / 'webroot').rglob('*')):
        if path.is_file() and not path.is_symlink() and path.suffix != '.mjs':
            files[path.relative_to(ROOT).as_posix()] = path.read_bytes()
    if 'webroot/app.js' not in files:
        raise ValueError('Missing WebUI bundle: run npm ci && npm run build:web before packaging')
    metadata = dict(line.split('=', 1) for line in files['module.prop'].decode().splitlines() if '=' in line)
    if metadata.get('id') != 'sortify':
        raise ValueError(f'Module identity must remain sortify, got {metadata.get("id")!r}')
    if not metadata.get('versionCode', '').isdecimal() or int(metadata['versionCode']) <= 10:
        raise ValueError(f'Module versionCode must be an integer greater than 10: {metadata.get("versionCode")!r}')
    files['SHA256SUMS'] = ''.join(f'{hashlib.sha256(data).hexdigest()}  {name}\n' for name, data in sorted(files.items())).encode()
    output.parent.mkdir(parents=True, exist_ok=True)
    temporary = output.with_suffix(output.suffix + '.tmp')
    checksum = output.with_name('SHA256SUMS')
    checksum_temporary = checksum.with_suffix('.tmp')
    try:
        with zipfile.ZipFile(temporary, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as package:
            for name, data in sorted(files.items()):
                item = zipfile.ZipInfo(name, (2026, 11, 15, 0, 0, 0))
                item.create_system = 3
                item.external_attr = (0o100755 if name.endswith('.sh') or name.startswith('bin/') else 0o100644) << 16
                item.compress_type = zipfile.ZIP_DEFLATED
                package.writestr(item, data)
        temporary.replace(output)
        checksum_temporary.write_text(
            f'{hashlib.sha256(output.read_bytes()).hexdigest()}  {output.name}\n'
        )
        checksum_temporary.replace(checksum)
    except (OSError, zipfile.BadZipFile) as error:
        temporary.unlink(missing_ok=True)
        checksum_temporary.unlink(missing_ok=True)
        raise RuntimeError(f'Could not write module ZIP {output}: {error}') from error
    print(output)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary-dir', type=Path, default=ROOT / 'build' / 'bin')
    parser.add_argument('--output', type=Path, default=ROOT / 'dist' / 'Sortify-v5.0.zip')
    arguments = parser.parse_args()
    try:
        assemble(arguments.binary_dir, arguments.output)
    except (ValueError, OSError, RuntimeError) as error:
        print(f'Package error: {error}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
