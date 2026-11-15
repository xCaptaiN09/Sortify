# Sortify v5 implementation and verification

Date: 2026-11-15. Branch: feat/sortify-v5. Base: deb8759.
Status: v5.0 published after Cap confirmed manual sorting on the phone.
Release: https://github.com/xCaptaiN09/Sortify/releases/tag/v5.0

## Delivered

- Standard-library Go native helper for configuration, source classification,
  duplicates, file transfers, journal recovery, process locks and scheduling.
- Size/SHA-256/byte duplicate comparison, exclusive destination creation,
  available filenames and no-replace source capture. Source files changing
  during checks or known open for writing are deferred.
- Stable-file observations and optional one-day organization delay. Automatic
  sorting defaults off on fresh install, with a one-hour interval.
- Configurable source/organization/category folders within the selected storage
  root, category presets and toggles, ordered custom rules, exclusions and
  unknown-file fallback. Shared configuration is parsed as JSON data.
- Atomic validated settings, interval limits 60..604800 seconds, 1..365-day
  retention, newest 200 runs maximum, settings import/export and upgrade migration.
- Preview, live state, truthful per-file summaries, Action button, WebUI Sort Now,
  clean Stop, individual/whole-run Undo, persistent holds and explicit release.
- Offline light editorial WebUI based on both supplied references, responsive
  Home/Preview/History/Settings, advanced editors and minimal xCaptaiN09 footer.
- Latest visual direction replaces appearance switching with one light theme,
  open sections, thin rules, fading gradient, authored line art and flat colored
  result cells. Bodoni Moda and DM Sans are bundled with their font licenses.
- Classic script bundle fixes direct-file module CORS failure: normal mode shows
  an unsupported environment; explicit ?demo=1 loads a labeled preview.
- Shell lifecycle entry points, reproducible native builds, ZIP assembler and CI.

## Validation performed

- Go: 26 engine tests plus one native CLI/subprocess integration test pass with
  the race detector; go vet passes.
- Node: 15 WebUI model/bridge/demo tests pass. Syntax and formatting checks pass.
- Python: six packaging/shell boundary tests pass; all module shell scripts pass
  sh -n. No user device files were used by tests.
- Real browser-to-host-native integration: Preview preserves the source; Sort
  moves the exact content; whole-run Undo restores it; holds persist; invalid
  interval does not change native settings; valid settings save/export succeeds.
  Special filenames are rendered as text, not HTML.
- Browser layout checks pass for all four screens at 320, 390, 768 and 1280px.
  Separate UI checks also covered 350px. Unsupported environments disable write
  controls. Demo mode requires ?demo=1 and is explicitly labeled.
- The single-theme palette has at least 6.89:1 measured text contrast across
  checked text/surface pairs. Reduced-motion artwork and keyboard skip-link
  behavior pass in Chromium.
- Current single-theme checks cover 40 page/width/OS-preference combinations:
  Home/Preview/History/Settings at 320, 390, 768, 1280 and 1920px, under both OS
  preferences. Direct-file unsupported/demo states, font loading and absence of
  the old theme selector pass. Previous dual-theme checks are superseded.
- Fresh visual review matched the requested composition and identified two finish
  defects: overlapping sticky Save/Discard and font-glyph action arrows. Both
  were corrected. The follow-up disposition is ship for those two scored fixes;
  all six replacement captures were valid, with no fix-induced regression seen.
- Code review found and corrected source-capture/recovery races, unlocked daemon
  state writes, lost cancellation, duplicate recovery counts, blocking special-file
  opens, absent-module lifecycle handling and lost backend bridge error details.
  Final scoped re-review found no remaining defect in those changes.
- ARM64 Android PIE, ARMv7 static, x86_64 static and x86 static helpers compile.
  Packaging produces dist/Sortify-v5.0.zip (approximately 6.8 MiB), including all
  four helpers, offline assets and SHA256SUMS. Development files are excluded.

## Phone report and permission fix

Cap tested the first candidate on a phone with KernelSU-Next. Eleven screenshots
show successful installation, working WebUI callbacks and a running service.
Preview, manual sorting and Action fail while opening the source path:
`/storage/emulated/0/Download: permission denied`.

The path walker opened every ancestor for directory reading. A host fixture with
an execute-only parent reproduces the same failure even though the Downloads
folder and its files are readable. Traversal now uses Linux O_PATH handles for
parents, retaining O_NOFOLLOW and final-file access checks. All five regression
subcases pass: scan, file read, destination creation, denied final access and
symlink rejection. The native CLI/async test also traverses an execute-only parent.
The phone's exact failing ancestor was not observable from the old generic error;
new errors include it. Cap subsequently confirmed that manual sorting works on
the phone after disabling the one-day delay. The second screenshots show normal
completed runs with two skipped files while the delay is enabled, not failures.

The installer now captures configuration JSON privately, prints three short
status lines and retains an error log only on failure. It removes architecture
directories after selecting bin/sortify; the universal ZIP still contains all four.
The obsolete source-root sortify.conf is removed; installed-v4 migration remains.
README and module use a new matching banner; see banner-provenance.md.

No ADB device was attached. Cap confirmed installation, manager callbacks and manual file sorting on the
phone. Undo, duplicate routing, reboot scheduling and upgrade migration are not
independently phone-verified by this confirmation.
The O_PATH change and installer received scoped read-only review with no Critical
or Important findings. Host race tests, vet, six shell/package tests and native
browser integration pass; architecture helpers have been rebuilt.

References: [Linux open(2)](https://man7.org/linux/man-pages/man2/open.2.html),
[AOSP directory-handle pattern](https://android.googlesource.com/platform/system/vold/+/1edb6550abe80993c96d86d4130f9bbe71de1b98/Utils.cpp).

Source capture requires renameat2(RENAME_NOREPLACE). Unsupported filesystems fail
closed and retain source data. Shared-storage mode0700 is not assumed to guarantee
private staging. Known-writer inspection covers only readable procfs entries;
Android restrictions may hide writers. Stable observations and repeated content
checks improve download handling but cannot establish completion for every app.

## Release state

module.prop and the helper/UI identify version 5.0/versionCode 11; id remains
sortify. The previous public release was v4.0/code 10. Release v5.0 targets
source commit cca0a7059445aa2b7c89e22779829c1d63f0c516. The uploaded ZIP was
downloaded and its SHA-256 and internal file checksums verified before publication.
The public release and tagged changelog were verified before advancing update.json.
GitHub Actions run 43719414015 passed every verification/build/package step.
The four freshly built Go 1.27.4 helpers match the phone-tested binary bytes.
Release ZIP SHA-256: 3be8834cf74f6c5e8e332fe781d47f95e15c2011d663692cfea6c901e18a2329.

## Artifacts

- build/sortify-mobile.png: fresh configuration, one-day delay on.
- build/sortify-preview-mobile.png: real temporary-file preview.
- build/sortify-settings-mobile.png: advanced settings.
- build/sortify-desktop.png: actual host-native sorting/Undo result.
- build/design-home-390.png, design-home-1280.png and design-home-1920.png:
  current single-theme, explicitly labeled demo views.
- build/design-settings-390.png, design-settings-1280.png and
  design-settings-1920.png: current settings views.
- dist/Sortify-v5.0.zip: v5.0 install ZIP; manual sorting confirmed by Cap on the phone.

Build commands and usage are in README.md. Approved scope is in NEXT_UPDATE.md.
The implementation plan is in plans/2026-11-15-sortify-v5.md.
