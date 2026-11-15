# Sortify

Sortify is a root-manager module that organizes downloads on Android. It pairs
a native Go helper with manager Action/service entry points and an offline WebUI.

The primary user wants to preview and organize files without losing collisions,
moving active downloads, or silently discarding failures. Automatic sorting is
off on fresh installs; its default interval is one hour. The optional one-day
delay starts enabled. Manual sorting also follows delay, ignore and Undo rules.

Home shows service/run status, manual sorting and scheduling. Preview explains
proposed destinations and skip reasons. History supports individual and whole-run
Undo plus persistent protection. Settings controls folders, categories, rules,
exclusions, retention, unknown-file handling and settings import/export.

The WebUI uses a supported KernelSU-family callback bridge. A normal browser
cannot access Android files; explicit `?demo=1` is an isolated, labeled simulation.
Assets and fonts are local. The author credit is `xCaptaiN09`.

Version 5.0/code 11 retains module ID sortify. Cap confirmed installation,
KernelSU-Next WebUI callbacks and manual sorting on the phone. Host tests also
cover Undo, duplicates, file preservation and settings. Those checks do not
establish phone verification of every feature or manager.

The latest visual brief and references are recorded in docs/design-direction.md.
Technical scope and limitations are in README.md and docs/IMPLEMENTATION_STATUS.md.
