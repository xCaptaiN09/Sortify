"""Browser-to-native integration in temporary storage, never real device folders.

Run with a Playwright-enabled Python after tools/build.py --host-only.
"""
from __future__ import annotations

import base64
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import shlex
import subprocess
import tempfile
import threading
import time

from playwright.sync_api import expect, sync_playwright

ROOT = Path(__file__).resolve().parents[1]
BINARY = ROOT / 'build/bin/host/sortify'


def main() -> None:
    with tempfile.TemporaryDirectory(prefix='sortify-browser-') as temporary:
        base = Path(temporary)
        storage = base / 'storage'
        source = storage / 'Download'
        source.mkdir(parents=True)
        fixture = source / 'invoice <sample> café.pdf'
        fixture.write_bytes(b'Sortify browser integration document')
        os.utime(fixture, (time.time() - 172800, time.time() - 172800))
        command = [str(BINARY), '--state', str(base / 'state'), '--storage', str(storage), '--module', str(ROOT)]
        subprocess.run(command + ['init'], check=True, capture_output=True)

        def rpc(request: dict) -> dict:
            encoded = base64.b64encode(json.dumps(request).encode()).decode()
            result = subprocess.run(command + ['request', encoded], text=True, capture_output=True, timeout=30, check=False)
            response = json.loads(result.stdout)
            if not response.get('ok'):
                raise RuntimeError(f'Native request {request["command"]}: {response.get("error")}')
            return response['data']

        settings = rpc({'command': 'snapshot'})['settings']
        rpc({'command': 'preview'})
        state_file = base / 'state/state.json'
        state = json.loads(state_file.read_text())
        for observation in state.get('observations', {}).values():
            observation['since'] = int(time.time()) - 120
        state_file.write_text(json.dumps(state))

        class Handler(SimpleHTTPRequestHandler):
            def __init__(self, *args, **kwargs):
                super().__init__(*args, directory=str(ROOT / 'webroot'), **kwargs)

            def log_message(self, *_args) -> None:
                pass

            def do_POST(self) -> None:
                if self.path != '/__native_test_bridge':
                    self.send_error(404)
                    return
                length = int(self.headers.get('Content-Length', '0'))
                if length <= 0 or length > 200000:
                    self.send_error(400)
                    return
                request = json.loads(self.rfile.read(length))
                parts = shlex.split(request['command'])
                if len(parts) != 4 or parts[:3] != ['sh', '/data/adb/modules/sortify/sortify.sh', 'request']:
                    self.send_error(400)
                    return
                result = subprocess.run(command + ['request', parts[-1]], text=True, capture_output=True, timeout=30, check=False)
                data = json.dumps({'errno': result.returncode, 'stdout': result.stdout, 'stderr': result.stderr}).encode()
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.send_header('Content-Length', str(len(data)))
                self.end_headers()
                self.wfile.write(data)

        server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        url = f'http://127.0.0.1:{server.server_port}'
        errors: list[str] = []
        try:
            with sync_playwright() as playwright:
                browser = playwright.chromium.launch(headless=True)
                page = browser.new_page(viewport={'width': 390, 'height': 844}, device_scale_factor=1)
                page.on('pageerror', lambda error: errors.append(str(error)))
                page.add_init_script('''window.ksu = {exec(command, options, callback) {
                  fetch('/__native_test_bridge', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({command})})
                    .then(r=>r.json()).then(r=>window[callback]?.(r.errno,r.stdout,r.stderr))
                    .catch(e=>window[callback]?.(1,'',e.message));
                }};''')
                page.goto(url, wait_until='networkidle')
                page.get_by_text('xCaptaiN09', exact=True).wait_for()
                page.screenshot(path=str(ROOT / 'build/sortify-mobile.png'), full_page=True)
                settings['delayEnabled'] = False
                rpc({'command': 'settings.save', 'settings': settings})
                page.get_by_role('button', name='Refresh', exact=True).click()
                assert page.evaluate('document.documentElement.scrollWidth <= innerWidth'), 'Mobile page overflows horizontally'
                destination = storage / 'Sortify/Documents' / fixture.name
                page.get_by_role('link', name='Preview', exact=True).click()
                page.get_by_role('button', name='Scan preview', exact=True).click()
                expect(page.locator('#preview-list')).to_contain_text(fixture.name)
                assert fixture.exists() and not destination.exists(), 'Preview moved the fixture'
                assert page.locator('#preview-list sample').count() == 0, 'Filename was interpreted as markup'
                page.screenshot(path=str(ROOT / 'build/sortify-preview-mobile.png'), full_page=True)
                page.get_by_role('link', name='Home', exact=True).click()
                page.get_by_role('button', name='Sort now', exact=True).click()
                deadline = time.monotonic() + 8
                while fixture.exists() and time.monotonic() < deadline:
                    time.sleep(0.1)
                assert not fixture.exists() and destination.read_bytes() == b'Sortify browser integration document', 'Real native sort did not preserve contents'
                page.get_by_role('button', name='Refresh', exact=True).click()
                page.get_by_role('link', name='History', exact=True).click()
                expect(page.locator('#history-list')).to_contain_text('Complete')
                page.locator('#history-list summary').first.click()
                page.get_by_role('button', name='Undo entire run', exact=True).first.click()
                page.locator('#confirm-accept').click()
                deadline = time.monotonic() + 8
                while not fixture.exists() and time.monotonic() < deadline:
                    time.sleep(0.1)
                assert fixture.read_bytes() == b'Sortify browser integration document' and not destination.exists(), 'Real native Undo did not restore the source'
                page.get_by_role('link', name='Home', exact=True).click()
                page.get_by_role('button', name='Refresh', exact=True).click()
                page.get_by_role('link', name='History', exact=True).click()
                expect(page.locator('#holds-list')).to_contain_text(fixture.name)
                page.get_by_role('link', name='Settings', exact=True).click()
                page.locator('#interval').fill('0')
                page.get_by_role('button', name='Save settings', exact=True).click()
                expect(page.locator('#notice')).to_contain_text('whole number')
                assert rpc({'command':'snapshot'})['settings']['intervalSeconds'] == 3600, 'Invalid UI input changed native settings'
                page.locator('#interval').fill('7200')
                page.get_by_role('button', name='Save settings', exact=True).click()
                expect(page.locator('#notice')).to_contain_text('Settings saved')
                assert rpc({'command':'snapshot'})['settings']['intervalSeconds'] == 7200, 'UI settings were not persisted by native backend'
                page.get_by_role('button', name='Export settings', exact=True).click()
                expect(page.locator('#settings-json')).not_to_have_value('')
                exported = json.loads(page.locator('#settings-json').input_value())
                assert exported['intervalSeconds'] == 7200
                page.locator('#advanced').evaluate('(node)=>node.open=true')
                page.screenshot(path=str(ROOT / 'build/sortify-settings-mobile.png'), full_page=True)
                for width in [320, 390, 768, 1280]:
                    page.set_viewport_size({'width':width, 'height':844})
                    for screen in ['Home','Preview','History','Settings']:
                        page.get_by_role('link', name=screen, exact=True).click()
                        expect(page.locator('#page-' + screen.lower())).to_be_visible()
                        if not page.evaluate('document.documentElement.scrollWidth <= innerWidth'):
                            page.screenshot(path=str(ROOT / 'build/overflow.png'), full_page=True)
                            print(page.evaluate('({inner:innerWidth,doc:document.documentElement.scrollWidth,client:document.documentElement.clientWidth,body:document.body.scrollWidth})'))
                            print(page.evaluate('Array.from(document.querySelectorAll("body *")).filter(e=>e.getBoundingClientRect().right>innerWidth+1).map(e=>({tag:e.tagName,id:e.id,class:e.className,right:e.getBoundingClientRect().right,text:e.textContent.slice(0,100)})).slice(0,20)'))
                            raise AssertionError(f'{screen} overflows at {width}px')
                page.get_by_role('link', name='Home', exact=True).click()
                page.screenshot(path=str(ROOT / 'build/sortify-desktop.png'), full_page=True)
                assert not errors, errors
                unsupported = browser.new_page(viewport={'width':390,'height':844})
                unsupported.goto(url, wait_until='networkidle')
                expect(unsupported.locator('#connection')).to_contain_text('Device access unavailable')
                expect(unsupported.get_by_role('button', name='Sort now', exact=True)).to_be_disabled()
                demo = browser.new_page(viewport={'width':390,'height':844})
                demo.goto(url + '/?demo=1', wait_until='networkidle')
                expect(demo.locator('#demo-banner')).to_be_visible()
                print('Passed: real native Preview, Sort, whole-run Undo, persistent holds, setting validation/save/export; 16 layout checks; unsupported mode and explicit demo.')
                browser.close()
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)


if __name__ == '__main__':
    main()
