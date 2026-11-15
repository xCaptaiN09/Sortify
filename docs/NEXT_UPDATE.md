# Sortify — next update requirements

Recorded: 2026-11-15.
Current baseline: v4.0, main commit deb8759.
Status: approved requirements, implemented on local feat/sortify-v5 branch;
release-candidate validation and device verification are recorded separately in
IMPLEMENTATION_STATUS.md. Cap authorized the v5.0 release after confirming manual
sorting on the phone. Version 5.0/versionCode 11 is published; module ID remains
sortify. Release: https://github.com/xCaptaiN09/Sortify/releases/tag/v5.0
Cap supplied two WebUI visual references
on 2026-11-15; the proposed direction is recorded in section 8.

## 1. Scope requested by Cap

Checked items are implemented locally; this is not a claim of device verification
or a published release. See IMPLEMENTATION_STATUS.md for validation limits.

- [x] Preserve every file when destination names collide.
- [x] Detect identical contents even when filenames differ.
- [x] Handle collisions inside Duplicates without overwriting anything.
- [x] Improve handling of incomplete and actively written downloads.
- [x] Prevent simultaneous scheduled, manual, WebUI and undo operations.
- [x] Report failures accurately instead of always saying completed.
- [x] Validate settings in the WebUI and backend.
- [x] Read configuration as data rather than executing it as shell code.
- [x] Scan Downloads once per run and classify each file from that scan.
- [x] Redesign the WebUI to be modern, clean and easy to use.
- [x] Show service status and the last sorting run.
- [x] Add Sort Now inside the WebUI.
- [x] Allow automatic interval sorting to be turned off while retaining manual sorting.
- [x] Add a preview of proposed moves.
- [x] Allow categories to be enabled or disabled.
- [x] Add ignore rules for selected files and filename patterns.
- [x] Add move history and Undo.
- [x] Retain the manager Action button for immediate sorting of eligible files.
- [x] Add custom destination folders and editable rules in Advanced settings.
- [x] Allow custom categories, extension lists, category destinations and toggles.
- [x] Change the default automatic sorting interval to one hour (3600 seconds).
- [x] Start with automatic sorting OFF on fresh installations (approved by Cap).
- [x] Expand format coverage and offer Ebooks, Fonts, Subtitles, Code/config and
  Torrents as optional category presets (approved by Cap).
- [x] Add a one-day organization delay enabled by default, with an off switch.
- [x] Add settings export/import and Undo for an entire run.
- [x] Add unknown-file handling: move to Others by default, or leave in the source.
- [x] Publish verified v5.0 assets and matching update metadata; versionCode 11,
  unchanged module ID sortify. Manual sorting confirmed by Cap on the phone.
- [x] Protect files restored by Undo from being automatically sorted again.
- [x] Add Stop current run, stopping between files after the current operation completes.
- [x] Warn about overlapping custom rules and show the winning rule.
- [x] Add configurable history retention with clear loss-of-Undo information.
- [x] Add a minimal xCaptaiN09 credit at the bottom of the WebUI.
- [x] Follow the latest single light theme direction; remove appearance modes,
  outlined cards and boxed fieldsets; add line art, fine rules and fading gradients.
- [x] Use stronger soft gradients and pastel color blocks based on the references,
  balancing a slightly playful look with clear settings and controls.

The sections below describe recommended behavior. Details explicitly marked as
proposals remain open for discussion; they are not additional approved features.

## 2. Filenames and real duplicates

A matching filename is a naming conflict, not proof of identical contents.
A real duplicate means matching file contents, irrespective of filename.
Never automatically delete files or use forced overwrites to resolve either case.

### Recommended behavior

| Situation | Result |
| --- | --- |
| Same filename, different contents | Keep both in the normal category. Rename the incoming file, for example report (1).pdf. |
| Different filenames, identical contents | Keep the existing copy in place. Move the incoming copy to Duplicates, recording which file it matches. |
| Same filename, identical contents | Keep the existing copy in place. Move the incoming copy to Duplicates under a free name. |
| Same name already exists in Duplicates | Allocate another free name, for example report (2).pdf; preserve all existing files. |
| Incoming file matches a file already in Duplicates | Preserve both copies there under separate names; do not delete either. |
| Same size but different contents | Treat as different files. Size alone never establishes duplication. |
| Content comparison fails or the source changes during checking | Leave the source in Downloads and report the reason. |

Preserve the original extension when allocating a name. Existing numbered names,
names without extensions and filesystem case-equivalent names must not cause an
overwrite. Do not rely only on checking a path before a forced move: destination
creation must refuse replacement if another process creates that path meanwhile.

### Comparing contents

1. Find candidate matches by file size.
2. Compare content hashes for same-size candidates.
3. Confirm matching candidates byte-for-byte before labeling them duplicates.
4. Check source stability before and after reading. Changed files are deferred.

Recommended search scope: Sortify's managed category folders and Duplicates,
including files that existed before the update and files moved earlier in the
current run. Do not scan the whole phone or move existing organized copies as a
side effect. An ignored file or disabled category stays in Downloads before any
duplicate routing is considered.

A maintained comparison index is a possible optimization. A stale index must
never be trusted after a file changes, disappears or is replaced. Establish
correct behavior first; measure hashing costs on large files before optimizing.

## 3. Unfinished downloads

- Continue skipping hidden files and known temporary download suffixes.
- Expand and normalize the suffix checks so case variants are handled.
- Require a minimum time since the file was last modified.
- Observe size and modification time across separate checks. Defer newly seen
  files until stability has been observed.
- Recheck immediately before a move and after content comparison.
- Apply the same rules to automatic sorting and Sort Now.
- Show a simple skip reason, such as Still changing or Too recent.

The minimum age and observation delay are design choices to settle before coding.
Neither an unchanged size nor an old timestamp proves download completion: a
paused download can look stable. Investigate available Android download-state
signals during implementation, and document their coverage. Do not claim that
heuristics detect every active download.

### One-day organization delay requested by Cap

Enable a 24-hour delay by default and provide a setting to turn it off. This is
separate from the short stability checks above: disabling the one-day delay must
not disable unfinished-download protection. Both the manager Action button and
WebUI Sort Now start a run immediately but honor eligibility, delay, ignore rules
and category settings; they must explain why a recent file was skipped.

Label the feature as Keep recent downloads for 1 day. Where download completion
cannot be observed, define eligibility using available file modification times
plus independent stability observations, and explain that this is an estimate,
not a guaranteed timestamp for completion. Do not use preserved creation times as
proof that a newly seen file finished downloading a day earlier. Source changes
reset stability observations and re-evaluate the delay.

## 4. One operation at a time

Use one shared lock for scheduled sorting, the manager Action button, WebUI Sort
Now and Undo. A second request reports Already running and does not start another
operation. Preview must either use a consistent snapshot or report that a run is
in progress. Handle stale locks after process termination or reboot without
mistaking a reused process ID for the owner.

The lock coordinates Sortify. Other apps can still change files, so move-time
checks and refusal to overwrite remain necessary.

### Stop current run

Provide a Stop control while sorting is active. The backend records the request
and checks it between files. Finish the current file's move and history record,
then release the lock without starting the next file. Do not kill the process or
interrupt an in-progress copy, move or final history write. Show Stopping while
waiting, then Stopped with the partial run's actual counts. Previously completed
moves remain available in history and Undo. Clear stale stop requests so they
cannot accidentally cancel a later run.

Stopping the current run does not turn off future scheduled sorting. Keep Stop
and the automatic sorting toggle separate and explain this in the UI.

## 5. Accurate results and failures

- Check directory creation, scanning, comparison, moving and state/history writes.
- Count moved, duplicates, renamed, skipped and failed files with clear definitions.
- Report Complete, Completed with errors, Failed or Already running accurately.
- Show a per-file reason for failures. Failed source files remain available.
- Distinguish permission errors, unavailable storage, destination conflicts,
  changing files and insufficient space when the underlying tools expose them.
- Recover sensibly if a run is interrupted. Record move intent and completion so
  history does not falsely claim a failed move succeeded, or silently omit a move
  that completed just before interruption.
- Keep logs bounded and provide one consistent history/status view.
- Allow users to configure history retention. Explain before applying a shorter
  retention period or clearing history that removed records are no longer
  available for Undo. Removing records must never remove the organized files.
  Retain unfinished operation records until recovery completes; do not prune a
  run while it is being read for Undo. Exact retention choices remain a design
  detail; use bounded storage rather than unlimited growth by default.
- Check manager command exit status before displaying a success message.

Example: Moved 12 files, skipped 3 unfinished downloads, failed to move 1 file.

## 6. Configuration and scheduling

- Parse an explicit set of supported settings; never source the configuration.
- Validate saved values in the backend as well as in the WebUI.
- Interval must be a bounded positive integer. Reject zero, negative values,
  fractions, exponent notation, non-numeric input and values outside the supported
  range. Exact minimum and maximum remain design choices.
- Invalid saves explain the problem and leave the previous working config intact.
- Write configuration atomically so a service cannot read a half-written file.
- Separate automatic sorting enabled/disabled from the interval value. Zero does
  not mean disabled.
- Saving an interval or turning off automatic sorting must wake/reload the
  scheduler rather than wait out its previous sleep. Turning auto sort off blocks
  future scheduled runs; it does not abruptly interrupt an in-progress file move.
- Manual sorting remains available when automatic sorting is off.
- Default interval: 3600 seconds (one hour), as requested by Cap. This controls
  scan frequency; the one-day delay controls when an individual file is eligible.
- Approved by Cap: fresh installations start with automatic sorting OFF until
  the user enables it. Offer Preview before enabling it. When enabled,
  use the configured hourly interval. Do not silently replace an existing user's
  saved enablement or interval during an upgrade.
- Preserve existing compatible settings during module upgrades.
- Avoid constructing shell commands from unchecked filenames or user input.

## 7. Sorting pipeline

Scan direct files in Downloads once per run. For each file: apply ignore rules,
check category enablement and download stability, classify the extension, compare
duplicate candidates and perform a move that cannot overwrite an existing file.

Do not change recursive behavior silently: Downloads subfolders remain untouched
unless a separate future feature explicitly enables them. Support spaces,
quotes, Unicode, leading dashes and newlines in filenames. Avoid depending on
newline-delimited filenames or injecting filenames into command text.

## 8. WebUI

Cap supplied these local references on 2026-11-15:

- /home/captain/Pictures/Screenshots/Screenshot_2026-11-15_12.29.27.png
- /home/captain/Pictures/Screenshots/Screenshot_2026-11-15_12.31.01.png

Both were visually inspected. They use light editorial layouts, prominent serif
type, crisp dark text, fine dividers, geometric line drawings and pastel color
fields. They are visual references, not screens to copy literally into a utility.

### Proposed visual direction

Updated by Cap on 2026-11-15: the rounded, outlined dashboard and theme switcher
were rejected. The current revision uses one light theme with open sections,
fine rules, a page-level fading gradient, Bodoni Moda/DM Sans, larger authored
line art and flat colored result cells. This replaces the earlier request for
System/Light/Dark modes. Assets and fonts are bundled for offline use.

- Use an off-white base, near-black text and lavender as the main accent, with
  restrained green for status. The reference's soft gradient can appear subtly
  in the header; controls must remain easy to read without relying on it.
- Use a bold sans-serif Sortify wordmark and serif section headings paired with clear sans-serif
  labels and body text. Bundle any custom fonts locally; the module stays offline.
- Prefer flat sections, thin dividers and generous spacing over nested cards,
  large shadows or excessive rounding. Keep a consistent modest corner radius.
- Use small line icons and strong type hierarchy. Reserve large numerals for
  meaningful counts or time values. Do not carry portraits, decorative giant
  headlines or poster-sized artwork into the working screens.
- Proposed navigation: Home, Preview, History and Settings. Advanced options
  remain inside Settings so the initial screen stays simple.
- Home prioritizes actual status, automatic sorting toggle, last result and
  Preview / Sort Now. Show Stop while a run is active. Detailed rules and paths
  belong in Settings, not the dashboard.
- Design for a narrow phone WebView first, with no horizontal scrolling. Use
  readable labels, sufficient contrast, at least 44px touch targets, visible
  focus states and clear busy/error/success feedback.
- Keep motion short and purposeful, with reduced-motion support. Do not use
  heavy animation or remote assets for a settings screen.
- Add the exact text xCaptaiN09 at the bottom as a small, muted, readable credit,
  separated by spacing or one fine divider. No badge, promotional block or large
  author logo. It should sit after content rather than obstruct controls.

Design read: an editorial mobile utility with straightforward controls, using the
supplied references' typography and pastel accents. Internal design dials:
variance 4/10, motion 2/10, density 4/10. This is a proposed interpretation of the
references, not an implemented UI or approval of a final mockup.

### Requested screens and controls

- Dashboard: automatic sorting on/off, actual run state, last run time and result.
- Sort Now: immediate run, busy feedback and the resulting summary.
- Stop current run: request a clean stop after the current file, show Stopping
  until acknowledged, then display the completed portion of the run.
- Settings: interval, category toggles and ignore rules.
- Basic settings: automatic sorting toggle, one-hour default interval, one-day
  delay toggle, common category controls and unknown-file behavior.
- Advanced settings: destination paths, custom categories, extension mappings,
  filename rules, rule ordering and ignore patterns. Display a readable rule
  summary and validate changes before saving.
- Preview: original filename, proposed destination, renamed result, duplicate
  match or skip reason. No file moves or history changes during preview. Explain
  that files may change before the actual run, which always checks them again.
- History: time, original path, destination, outcome and available Undo action.
- Undo: restore an unchanged moved file to its original path. If that path is
  occupied, or the moved file changed/disappeared, report a conflict and leave
  files intact. Never overwrite a newer file to restore an older one.
- Undo run: restore a selected run in reverse move order, using the same shared
  lock and per-file conflict checks. Show restored, conflicted and failed counts;
  never present partial success as a fully restored run.
- Undo protection: register each successfully restored file before releasing the
  operation lock. Mark it as held so automatic sorting cannot immediately move
  it again. Keep holds across reboots and independently of prunable history;
  clearing history must not silently remove the protection. Show Restored by Undo
  as the skip reason, with an explicit Allow sorting again control. Normal Sort
  Now and Action runs honor the same hold. Do not silently expire holds after a
  short delay or treat a reused filename as proof it is the same restored file;
  validate identity/content before applying a hold to a changed or replaced file.
- Export/import settings: export a versioned settings document; validate the
  complete import before applying it atomically. Import rules as data, never
  commands. Do not import move history, process locks or content indexes as
  settings. Show a summary and verify folder access on the new device.

Disabling a category leaves those files in Downloads; it must not send them to
Others or Duplicates. Unknown extensions move to Others by default, with an option
to leave them in the source. If Others is disabled, those files stay in the source.
Start with simple filename/glob ignore rules; advanced regular expressions are
not required.

Verify the actual WebUI bridge against intended managers before selecting its
API. Handle missing or incompatible bridges visibly. Preserve the terminal and
manager Action entry points so basic sorting does not depend on the WebUI.

## 9. Advanced paths, categories and rules

### Custom folders

- Let users select a custom organization root instead of /sdcard/Sortify.
- Let a category use a custom destination underneath or outside that root within
  accessible shared storage. Duplicate comparison covers explicitly managed
  destinations; it does not expand into unrelated folders.
- Custom source-folder selection is a further proposal: default remains
  /sdcard/Download. If adopted, start with one source rather than adding multiple
  simultaneous watched folders in the first release.
- Resolve and validate folder paths, permissions, aliases and symlinks. Reject
  equal source/destination paths, source/destination nesting and destinations in
  protected system or module directories. Do not let rules repeatedly sort their
  own outputs. Never execute path text as a command.
- Creating a custom category does not silently reorganize existing destination
  files. Preview what happens to eligible new files from the source.
- Across different filesystems, a move may need copying. Retain the source until
  a complete, verified destination is committed without replacement. Handle
  insufficient space and interruption without losing the source.

### User-defined categories and rules

- Create, rename, enable/disable and remove custom categories. Removing a rule or
  category must not delete its files. Category labels and destination paths are
  separate settings.
- Edit supported extensions for built-in or custom categories, case-insensitively.
- Match filename patterns and extensions, including compound suffixes such as
  tar.gz. Give more specific suffixes precedence when appropriate.
- Recommended priority: incomplete-file exclusions, user ignore rules, ordered
  custom rules, built-in extension mappings, then the unknown-file fallback.
  Show which rule matched in Preview. First matching custom rule wins.
- Warn when enabled custom rules overlap. Flag identical mappings and obvious
  overlaps when saving; explain that arbitrary glob overlaps may not all be
  detected statically. In Preview, show all custom rules actually matching each
  file and identify the first matching rule as the winner. Keep rule order
  visible and editable. Do not describe every overlap as an error: deliberate
  specific-before-general rules are valid.
- A rule matching a disabled category leaves that file in the source; it must not
  fall through into another category or Others.
- Custom rules change classification/destination only. They cannot bypass
  unfinished-download checks, duplicate protection, the shared lock or Undo rules.
- Begin with extension lists and simple filename/glob patterns. Arbitrary shell
  commands and scripts are not a rule feature. Regex or content/MIME matching
  would require a later explicit scope decision.

### Format coverage and optional category presets

Cap approved expanding format coverage and adding the optional category presets.
The list below records candidate mappings, including the approved examples; final
mapping details must resolve ambiguous extensions through explicit rule priority.
Extension alone does not identify file contents. Users can add or override
mappings in Advanced.

| Category | Additional extensions to consider |
| --- | --- |
| Documents | odt, ods, odp, rtf, md, log, tsv |
| Images | avif, tif, tiff, ico, dng, cr2, nef, arw |
| Videos | m4v, ts, mts, m2ts, vob, wmv |
| Audio | alac, aiff, aif, ape, amr, mka, mid, midi |
| Archives | xz, zst, tgz, tbz2, txz, lz4 |
| Apps/installers | apks, apkm, xapk, aab, msi, deb, rpm, AppImage |
| Ebooks — optional new category | epub, mobi, azw, azw3, cbz, cbr |
| Fonts — optional new category | ttf, otf, woff, woff2 |
| Subtitles — optional new category | srt, ass, ssa, vtt, sub |
| Code/config — optional new category | py, sh, js, ts, tsx, jsx, c, h, cpp, java, kt, json, yaml, yml, xml, toml, ini, html, css |
| Torrents — optional new category | torrent |

New categories should be offered as presets rather than silently creating many
new folders. For example, JSON can remain in Others until the user enables a
Code/config preset or adds a custom rule. PDF stays in Documents by default;
users can override that with a more specific custom rule for their ebooks.
Files without extensions or without any enabled mapping use the Others fallback,
unless ignored. Other download pieces or partial multi-part archives must not be
treated as complete just because an extension appears in this table.

## 10. Additional ideas for discussion

These are suggestions, not approved update scope:

- Reset settings to defaults without touching files.
- Optionally limit large automatic runs or postpone heavy hashing on low battery.
  Measure real cost before adding scheduling complexity.

Do not add automatic duplicate deletion, cloud uploads, AI classification,
notifications, recursive sorting or filesystem watchers without a separate scope
decision. They are unnecessary for the requested first update.

## 11. Verification before release

- Test all duplicate/name-conflict cases in section 2, including repeated runs.
- Test files changing during observation, hashing and immediately before moving.
- Test simultaneous service/Action/WebUI/Undo requests and stale-lock recovery.
- Test missing storage, permission errors, interrupted moves and logging failures.
- Test invalid settings, interrupted saves and automatic-sort disable/reload.
- Test the one-hour interval and 24-hour delay independently, including manual
  Action, delay disabled, reboots and preserved timestamps on incoming files.
- Test custom paths, overlapping source/destination trees, cross-filesystem moves,
  imported paths and permissions. Test rule priority and disabled custom categories.
- Test settings import rejection and whole-run Undo with partial conflicts.
- Confirm excluded files and disabled categories stay in Downloads.
- Confirm preview makes no moves and execution revalidates its proposals.
- Test successful Undo, occupied original paths and changed/missing moved files.
- Test Undo holds across scheduled/manual runs, reboot, history pruning and
  file replacement. Verify explicit release makes the file eligible again.
- Test Stop during hashing/copying/moving, accurate partial summaries, lock
  release, stale stop requests and the next scheduled run.
- Test rule overlap warnings, reordered rules and per-file winning-rule display.
- Test retention changes and clearing history: no files removed, no unfinished
  records pruned and no Undo offered for records that no longer exist.
- Test special filenames, large files and comparison across existing folders.
- Test real Android shell/tool behavior and intended manager WebUI bridges.
- Test install, upgrade, reboot, disable and uninstall lifecycle. Organized files
  and history must not be silently destroyed.
- Update README with actual behavior, limitations and supported managers after
  implementation is tested. Select release version and metadata at release time.

## 12. Update identity and release metadata

- Keep id=sortify unchanged. This identifies the installed module; changing it
  would create a different identity rather than correctly upgrade this module.
- Current metadata is version=4.0 and versionCode=10. Select a higher release
  version and a strictly higher integer versionCode when preparing the release.
  Version 5.0 with versionCode 11 is a proposal, not a released or reserved value.
  Recheck current published metadata before choosing the final number.
- Keep module.prop, update.json, the release tag, README and the actual release
  ZIP consistent. The module.prop inside the ZIP must contain the new version.
- Keep the existing updateJson endpoint so installed users can discover updates.
  Publish and verify the ZIP and changelog URLs before announcing the update.
  Use a supported changelog URL rather than putting inline prose in a URL field.
- Test upgrade from v4.0: preserve files, existing settings where compatible and
  module identity. Explain new delay defaults and any migrated behavior.
- Verify an installed older manager/module can detect and install the newer
  version; a source commit alone does not publish a module update.

References: [KernelSU module guide](https://kernelsu.org/guide/module.html) and
[Magisk module guide](https://topjohnwu.github.io/Magisk/guides.html).

## 13. Suggested delivery order

1. File preservation, duplicate comparison, shared locking and failure reporting.
2. Download handling, validated configuration, custom paths/rules and the single-pass sorter.
3. Status, preview, history/Undo, import/export and scheduler controls.
4. WebUI implementation using the supplied references and agreed visual direction.
5. Device testing, upgrade checks, documentation and release preparation.

This document records approved scope and design decisions. Implementation is
tracked in IMPLEMENTATION_STATUS.md; saving these requirements does not commit,
push or publish a release.
