"""Test shell boundaries without touching Android storage."""
from pathlib import Path
import json
import os
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ShellTests(unittest.TestCase):
    def test_installer_failure_keeps_diagnostic_without_printing_json(self) -> None:
        with tempfile.TemporaryDirectory(prefix='sortify failed install ') as temporary:
            root = Path(temporary)
            module = root / 'module'
            state = root / 'state'
            helper = module / 'bin/arm64/sortify'
            helper.parent.mkdir(parents=True)
            helper.write_text('''#!/bin/sh
printf '%s\\n' '{"ok":false,"error":"permission denied in fixture"}'
exit 1
''')
            helper.chmod(0o755)
            for name in ('action.sh', 'service.sh', 'sortify.sh', 'uninstall.sh'):
                (module / name).touch()
            script = (ROOT / 'customize.sh').read_text().replace(
                'STATE=/data/adb/sortify', f'STATE={str(state)!r}'
            ).replace(
                'LEGACY_CONF=/data/adb/modules/sortify/sortify.conf',
                f'LEGACY_CONF={str(root / "absent-legacy.conf")!r}',
            )
            target = root / 'customize.sh'
            target.write_text(script)
            harness = '''
ui_print() { printf '%s\\n' "$*"; }
abort() { printf '%s\\n' "$*" >&2; exit 1; }
set_perm() { chmod "$4" "$1"; }
. "$SORTIFY_INSTALL_SCRIPT"
'''
            result = subprocess.run(
                ['sh', '-c', harness], text=True, capture_output=True, check=False,
                env={**os.environ, 'MODPATH': str(module), 'ARCH': 'arm64',
                     'SORTIFY_INSTALL_SCRIPT': str(target)},
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertNotIn('{"ok"', result.stdout + result.stderr)
            self.assertNotIn('Ready.', result.stdout)
            self.assertIn('Settings setup failed', result.stderr)
            self.assertIn('permission denied in fixture', (state / 'install.log').read_text())

    def test_installer_has_short_output_and_keeps_only_selected_helper(self) -> None:
        with tempfile.TemporaryDirectory(prefix='sortify install ') as temporary:
            root = Path(temporary)
            module = root / 'module'
            state = root / 'state'
            selected = module / 'bin/arm64/sortify'
            selected.parent.mkdir(parents=True)
            selected.write_text('''#!/bin/sh
printf '%s\\n' '{"ok":true,"data":{"schema":1,"categories":["large configuration"]}}'
''')
            selected.chmod(0o755)
            for arch in ('arm', 'x64', 'x86'):
                directory = module / 'bin' / arch
                directory.mkdir()
                (directory / 'sortify').write_text('unused helper')
            script = (ROOT / 'customize.sh').read_text().replace(
                'STATE=/data/adb/sortify', f'STATE={str(state)!r}'
            ).replace(
                'LEGACY_CONF=/data/adb/modules/sortify/sortify.conf',
                f'LEGACY_CONF={str(root / "absent-legacy.conf")!r}',
            )
            target = root / 'customize.sh'
            target.write_text(script)
            harness = '''
ui_print() { printf '%s\\n' "$*"; }
abort() { printf '%s\\n' "$*" >&2; exit 1; }
set_perm() { chmod "$4" "$1"; }
. "$SORTIFY_INSTALL_SCRIPT"
'''
            for name in ('action.sh', 'service.sh', 'sortify.sh', 'uninstall.sh'):
                (module / name).touch()
            result = subprocess.run(
                ['sh', '-c', harness], text=True, capture_output=True, check=False,
                env={**os.environ, 'MODPATH': str(module), 'ARCH': 'arm64',
                     'SORTIFY_INSTALL_SCRIPT': str(target)},
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertNotIn('"schema"', result.stdout)
            self.assertLessEqual(len(result.stdout.splitlines()), 6)
            self.assertTrue((module / 'bin/sortify').is_file())
            self.assertEqual(sorted(p.name for p in (module / 'bin').iterdir()), ['sortify'])

    def test_wrapper_initializes_state_and_forwards_arguments_without_execution(self) -> None:
        with tempfile.TemporaryDirectory(prefix='sortify shell ') as temporary:
            root = Path(temporary)
            module = root / 'module with space'
            (module / 'bin').mkdir(parents=True)
            shutil.copyfile(ROOT / 'sortify.sh', module / 'sortify.sh')
            helper = module / 'bin/sortify'
            helper.write_text('''#!/usr/bin/env python3
from pathlib import Path
import json, os, sys
arguments = sys.argv[1:]
with open(os.environ['SORTIFY_CAPTURE'], 'a') as output:
    output.write(json.dumps(arguments) + '\\n')
if 'init' in arguments:
    state = Path(arguments[arguments.index('--state') + 1])
    state.mkdir(parents=True, exist_ok=True)
    (state / 'config.json').write_text('{}')
print('{"ok":true,"data":{"forwarded":true}}')
''')
            helper.chmod(0o755)
            sentinel = root / 'must-not-exist'
            payload = f'$(touch "{sentinel}"); wildcard * quote \' newline\n'
            capture = root / 'arguments.jsonl'
            environment = {**os.environ, 'SORTIFY_STATE_DIR': str(root / 'state'),
                           'SORTIFY_STORAGE_ROOT': str(root / 'storage'),
                           'SORTIFY_CAPTURE': str(capture)}
            result = subprocess.run(['sh', str(module / 'sortify.sh'), 'request', payload],
                                    env=environment, text=True, capture_output=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(json.loads(result.stdout)['ok'])
            calls = [json.loads(line) for line in capture.read_text().splitlines()]
            self.assertEqual(len(calls), 2)
            self.assertEqual(calls[-1][-2:], ['request', payload])
            self.assertFalse(sentinel.exists())

    def test_missing_helper_returns_failure_not_success(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            script = Path(temporary) / 'sortify.sh'
            shutil.copyfile(ROOT / 'sortify.sh', script)
            result = subprocess.run(['sh', str(script), 'run'], text=True,
                                    capture_output=True, check=False)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(json.loads(result.stdout)['ok'])


if __name__ == '__main__':
    unittest.main()
