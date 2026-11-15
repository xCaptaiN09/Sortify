"""Verify the single-theme WebUI and direct-file preview without a native bridge."""
from __future__ import annotations

from pathlib import Path

from playwright.sync_api import expect, sync_playwright

ROOT = Path(__file__).resolve().parents[1]


def main() -> None:
    errors: list[str] = []
    checks = 0
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        for preference in ('light', 'dark'):
            context = browser.new_context(color_scheme=preference)
            context.add_init_script("localStorage.setItem('sortify.appearance', 'dark')")
            page = context.new_page()
            page.on('pageerror', lambda error: errors.append(str(error)))
            page.goto((ROOT / 'webroot/index.html').as_uri(), wait_until='load')
            expect(page.locator('#connection')).to_contain_text('Device access unavailable')
            expect(page.locator('#sort-now')).to_be_disabled()
            expect(page.locator('#theme-mode')).to_have_count(0)
            assert page.evaluate('getComputedStyle(document.documentElement).colorScheme') == 'light'
            page.goto((ROOT / 'webroot/index.html').as_uri() + '?demo=1', wait_until='load')
            expect(page.locator('#demo-banner')).to_be_visible()
            expect(page.locator('#sort-now')).to_be_enabled()
            expect(page.locator('#connection')).not_to_contain_text('Connecting')
            page.evaluate('document.fonts.ready')
            assert page.evaluate("document.fonts.check('16px \"Bodoni Moda\"')")
            assert page.evaluate("document.fonts.check('16px \"DM Sans\"')")
            for width in (320, 390, 768, 1280, 1920):
                page.set_viewport_size({'width': width, 'height': 1000})
                for screen in ('Home', 'Preview', 'History', 'Settings'):
                    page.get_by_role('link', name=screen, exact=True).click()
                    expect(page.locator('#page-' + screen.lower())).to_be_visible()
                    assert page.evaluate('document.documentElement.scrollWidth <= innerWidth'), (
                        f'{preference}/{width}/{screen}: horizontal overflow'
                    )
                    checks += 1
                if preference == 'light' and width in (390, 1280, 1920):
                    for screen in ('Home', 'Settings'):
                        page.get_by_role('link', name=screen, exact=True).click()
                        page.evaluate('window.scrollTo(0, 0)')
                        page.screenshot(
                            path=str(ROOT / f'build/design-{screen.lower()}-{width}.png'),
                            full_page=True,
                            animations='disabled',
                        )
            context.close()
        assert not errors, errors
        browser.close()
    print(f'Passed direct-file unsupported/demo states, local fonts and {checks} responsive layouts.')


if __name__ == '__main__':
    main()
