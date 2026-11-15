# Sortify changelog

## 5.0 — 2026-11-15

- Preserve filename collisions and compare duplicate contents.
- Add validated settings, custom rules, categories and destinations.
- Add stable-file checks and optional one-day organization delay.
- Coordinate automatic, manual, WebUI and Undo operations with one lock.
- Add preview, history, Undo protection and clean stop requests.
- Add offline editorial WebUI with status, settings import/export and author credit.
- Use one light theme with open sections, fading gradients, serif typography,
  authored line art and flat colored result cells; remove outlined dashboard cards.
- Bundle the WebUI script and fonts for offline use and direct-file previews.
- Fresh installs start with automatic sorting off and a one-hour interval.
- Fix shared-storage traversal through search-only parents and report the exact
  path component on failure, while preserving symlink and final-access checks.
- Keep installation output short, preserve failure diagnostics, and remove
  unneeded architecture copies after selecting the native helper.
- Replace the README/module banner to match the WebUI; remove obsolete sortify.conf.
- Refresh the external ZIP checksum whenever packaging.
- Keep module ID sortify; increase versionCode from 10 to 11.

Fresh installations start with the one-day delay enabled. Manual sorting follows
that delay too. To test sorting immediately, turn it off in Settings and save;
the 60-second stability check remains active. The one-day timer includes the time
since Sortify first observed an unchanged file, even for older downloads.

Manual sorting was confirmed on Cap's Android phone with KernelSU-Next. Host tests
cover Preview, sorting, whole-run Undo, holds, settings validation and export.
This does not claim phone verification of every feature or module manager.
