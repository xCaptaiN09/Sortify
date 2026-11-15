"""Exercise release assembly as an executable against an actual module tree."""
from pathlib import Path
import hashlib
import subprocess
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[1]


class PackageTests(unittest.TestCase):
    def test_package_refuses_missing_executables(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            result = subprocess.run(
                ['python3', str(ROOT / 'tools/package.py'), '--binary-dir', temporary,
                 '--output', str(Path(temporary) / 'release.zip')],
                text=True, capture_output=True, check=False,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Missing native helper', result.stderr)
            self.assertFalse((Path(temporary) / 'release.zip').exists())

    def test_package_contains_offline_module_not_development_files(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            base = Path(temporary)
            for arch in ['arm64', 'arm', 'x64', 'x86']:
                directory = base / 'bin' / arch
                directory.mkdir(parents=True)
                (directory / 'sortify').write_bytes(b'\x7fELFtest-fixture')
            output = base / 'Sortify.zip'
            result = subprocess.run(
                ['python3', str(ROOT / 'tools/package.py'), '--binary-dir', str(base / 'bin'),
                 '--output', str(output)], text=True, capture_output=True, check=False,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(
                (base / 'SHA256SUMS').read_text(),
                f'{hashlib.sha256(output.read_bytes()).hexdigest()}  {output.name}\n',
            )
            with zipfile.ZipFile(output) as package:
                names = package.namelist()
                self.assertIn('module.prop', names)
                self.assertIn('webroot/index.html', names)
                self.assertIn('webroot/app.js', names)
                self.assertIn('webroot/fonts/bodoni-moda.woff2', names)
                self.assertFalse(any(name.endswith('.mjs') for name in names))
                self.assertIn('bin/arm64/sortify', names)
                self.assertIn('sortify.sh', names)
                self.assertIn('SHA256SUMS', names)
                self.assertFalse(any(name.startswith(('.git/', 'tests/', 'core/')) for name in names))
                metadata = package.read('module.prop').decode()
                self.assertIn('id=sortify\n', metadata)
                self.assertIn('versionCode=11\n', metadata)
                self.assertFalse(any('node_modules' in name for name in names))


if __name__ == '__main__':
    unittest.main()
