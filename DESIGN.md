---
name: Sortify
description: A clear editorial utility for organizing Android downloads.
colors:
  paper: "#f4f5f0"
  ink: "#18231e"
  muted: "#4c574f"
  line: "#a5afa4"
  lavender: "#d4b4ef"
  green: "#abd4b7"
  peach: "#f2b58d"
  blue: "#c2d7f3"
  danger: "#8c3025"
  field-bg: "#e9ede4"
  field-rule: "#788978"
  hover: "#e1e7db"
  focus: "#395d40"
  error-bg: "#f1d8cd"
  error-ink: "#702a20"
typography:
  display:
    fontFamily: "Bodoni Moda, Georgia, serif"
    fontSize: "clamp(48px, 6.6vw, 88px)"
    fontWeight: 400
    lineHeight: 1.04
    letterSpacing: "-0.035em"
  headline:
    fontFamily: "Bodoni Moda, Georgia, serif"
    fontSize: "clamp(28px, 3vw, 36px)"
    fontWeight: 400
    lineHeight: 1.2
    letterSpacing: "-0.025em"
  title:
    fontFamily: "DM Sans, sans-serif"
    fontSize: "17px"
    fontWeight: 600
    lineHeight: 1.35
  body:
    fontFamily: "DM Sans, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.55
  label:
    fontFamily: "DM Sans, sans-serif"
    fontSize: "13px"
    fontWeight: 600
    lineHeight: 1.55
  small:
    fontFamily: "DM Sans, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.55
  metric:
    fontFamily: "Bodoni Moda, Georgia, serif"
    fontSize: "36px"
    fontWeight: 400
    lineHeight: 1.05
rounded:
  square: "0"
  switch: "20px"
  circle: "50%"
spacing:
  "8": "8px"
  "12": "12px"
  "16": "16px"
  "20": "20px"
  "24": "24px"
  "32": "32px"
  "40": "40px"
  "56": "56px"
  "64": "64px"
components:
  button-primary:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
    typography: "{typography.label}"
    rounded: "{rounded.square}"
    padding: "13px 20px"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.square}"
    padding: "10px 16px"
  button-secondary-hover:
    backgroundColor: "{colors.hover}"
  button-text:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    rounded: "{rounded.square}"
    padding: "4px 0"
  field:
    backgroundColor: "{colors.field-bg}"
    textColor: "{colors.ink}"
    rounded: "{rounded.square}"
    padding: "11px 12px"
  outcome-move:
    backgroundColor: "{colors.green}"
    textColor: "{colors.ink}"
    typography: "{typography.small}"
    rounded: "{rounded.square}"
    padding: "3px 8px"
  outcome-duplicate:
    backgroundColor: "{colors.lavender}"
    textColor: "{colors.ink}"
    typography: "{typography.small}"
    rounded: "{rounded.square}"
    padding: "3px 8px"
  outcome-error:
    backgroundColor: "{colors.error-bg}"
    textColor: "{colors.error-ink}"
    typography: "{typography.small}"
    rounded: "{rounded.square}"
    padding: "3px 8px"
  metric:
    textColor: "{colors.ink}"
    rounded: "{rounded.square}"
    padding: "16px 12px"
  switch:
    rounded: "{rounded.switch}"
    width: "42px"
    height: "24px"
---

# Design System: Sortify

## Overview

**Creative North Star: "A clear editorial utility"**

Sortify combines cool paper, near-black green ink and high-contrast serif headings with compact, readable controls. A fading color field and authored globe/file line art give the utility an identity; open sections and fine rules keep operational information clear. This records the implemented local candidate, not an Android-verified release.

The user pinned the BOLT/Twinkle references and replaced the earlier dashboard direction with one light theme. No concept roll, seed or app comp was generated or approved. The implementation is the token source; `PRODUCT.md` supplies product truth and `docs/design-direction.md` records the confirmed visual authority.

**Key Characteristics:**

- One light paper-and-ink theme, independent of operating-system appearance.
- Bodoni Moda headings and counts; DM Sans navigation, labels and controls.
- Open sections, thin dividers and square functional surfaces.
- Fading gradients and original inline line artwork.
- Flat result colors, explicit states and restrained motion.

## Colors

Muted botanical neutrals support pastel result colors; the darkest ink carries actions and reading hierarchy.

### Primary

- **Ink:** main text, primary actions, structural rules, wordmark and active switches. Primary action text uses Paper.

### Secondary

- **Lavender:** duplicate outcomes and result cells.
- **Green:** move outcomes and the dominant moved-result cell; renamed-result cells also use this color.
- **Peach:** skipped-result cells.
- **Blue:** failed-result counts. An actual error outcome uses Error Background and Error Ink instead; count color is not the error-message treatment.

### Neutral

- **Paper:** page, primary-action lettering and dialog surface.
- **Muted:** explanatory copy, timestamps, paths and hints.
- **Line:** subordinate row dividers; Ink separates major sections.
- **Field Background / Field Rule:** quiet filled inputs with one bottom rule.
- **Hover / Focus:** pale hover fill and dark green keyboard focus or link emphasis.
- **Danger:** destructive text actions and unavailable-status dot.
- **Error Background / Error Ink:** shared error notice and error-outcome treatment.

The wide-screen background blends pale green, lilac and periwinkle, fading vertically into Paper. The narrow-screen gradient uses its own softer stops. These multi-stop treatments remain in the page stylesheet rather than being flattened into a single accent token. Artwork has a small separate illustration palette; do not replace the shared operational colors with art fills.

**The Single Light Theme Rule.** Keep the paper-and-ink system under both operating-system appearance preferences. Do not add an automatic dark variant.

## Typography

**Display Font:** Bodoni Moda, with Georgia and serif fallbacks. The bundled display face is regular; headings and result counts use optical sizing (`"opsz" 18`).

**Body Font:** DM Sans, with sans-serif fallback. The local variable face supports weights 400–700. Both fonts are bundled for offline use.

**Character:** Delicate serif forms supply the editorial voice. Sans-serif controls carry dense functional detail without competing with headings. The JSON transfer textarea alone uses the platform monospace stack.

### Hierarchy

- **Display:** the large two-line Home heading; the normative desktop ramp is in frontmatter. Other pages use a smaller ramp (`clamp(44px, 6vw, 70px)`).
- **Headline:** section headings. Settings legends use the same serif character at a fixed size (28px).
- **Title:** file names and smaller headings; the last-run title steps down slightly (16px).
- **Body:** the inherited reading style; paragraphs are capped at 74ch. Page introductions use a smaller size (14px) and a 60ch maximum.
- **Label:** buttons and compact control copy. Field labels use medium weight (500); setting labels use a slightly larger semibold style (14px, 600).
- **Small:** paths, dates, helper text, navigation on small screens and result labels. Preserve wrapping rather than hiding necessary text.
- **Metric:** serif counts with tabular numerals. The main moved count is larger (56px); other mobile counts reduce to 32px.

**The Two Voices Rule.** Use the serif for headings, section legends, artwork annotation and counts. Use DM Sans for actions, navigation and explanations.

## Layout

The centered shell has a maximum outer width (1184px) and desktop side padding (40px). The masthead, horizontal navigation, page content and author footer share the same edges. Navigation remains four visible text links at every width; the mobile layout removes icons rather than introducing a menu.

Home places copy and artwork in a two-column hero (1.15fr / 1fr), then separates service status with horizontal rules. Its operating sections use two open columns (1.05fr / 1fr) with a generous gap (64px). Settings forms and transfer content narrow to 780px above the desktop breakpoint (900px), while their page heading remains aligned to the full shell.

At widths up to 760px, side padding becomes 24px and column gaps tighten. Up to 540px, side padding becomes 22px; Home becomes one column and the artwork occupies a reserved area above the heading. The hero reserves 168px before its copy, so the illustration does not cover text. Settings category grids and editor grids also become one column. Filter controls retain two columns with reduced spacing.

Most local gaps and padding follow the frontmatter spacing steps. Major transitions add open space rather than another container. Repeated settings rows and file rows use bottom rules. Fieldsets have only a top rule, interrupted by their serif legend.

Save/Discard stays in normal document flow after the settings sections. On narrow screens its dirty-state label occupies one row and its actions align right below it. It must not cover form controls. Long file paths and filenames wrap, and action groups wrap when needed.

## Elevation & Depth

No component uses a box shadow. Depth comes from the fading page gradient, flat color fields, the contrast between paper and ink, and the line artwork's overlapping forms. Dialogs use a dark translucent backdrop (`rgb(24 35 30 / 45%)`) and a strong top rule rather than a floating-card shadow. The dialog is bounded to the viewport and scrolls internally when needed.

**The Open Section Rule.** Separate routine content with space and rules. Do not reintroduce rounded or outlined dashboard cards around the operating sections.

## Shapes

Buttons, input surfaces, outcome tags, result cells and dialogs have square corners. Rounded forms are intentional exceptions: switch tracks and knobs, status dots, the circular history-link arrow, the globe and the annotation's curved underline. Do not extend the square-surface rule into a ban on the artwork's own circles and arcs.

Major boundaries are hairline ink rules (1px); subordinate rows use Line. The active navigation item adds a heavier underline (3px). Inline SVG action arrows share the same northeast path and stroke weight (1.5), inheriting the action's color. The illustration uses a finer stroke (1.2).

## Components

### Buttons

Primary actions are ink rectangles with Paper text, square corners and an optional inline SVG arrow. Secondary actions are transparent with a single bottom rule. Text actions remove that rule and use a conventional underline. Destructive variants use Danger text rather than another filled color.

Buttons have a minimum height (44px); text actions use 32px. Primary hover becomes a deeper botanical green. Secondary hover adds Hover fill; text hover changes ink color. Keyboard focus is a visible green outline (2px) offset from the component (5px). Pressed buttons move down by 1px. Disabled controls use reduced opacity (0.48) and a disabled cursor; do not make an unavailable action look enabled.

### Inputs / Fields

Inputs, selects and textareas use Field Background, one Field Rule bottom stroke and square corners. They have a minimum height (45px), generous inline padding and the same keyboard outline as actions. Labels sit above; hints stay below. Textareas resize vertically. Regular checkboxes remain native controls; binary setting switches use the custom track described below.

### Navigation

Four text links sit between ink rules. The current page is marked with a heavier underline and `aria-current="page"`. Desktop links include small inline outline SVGs; these disappear at the narrow breakpoint while labels stay visible. Hover changes the text to the focus green. Navigation is part of document flow.

### Outcome Tags and File Rows

Outcome tags are small flat rectangles, not pills. Move, duplicate and error use their established color roles; remaining outcomes use a muted neutral field. Filenames lead a row, with paths, reasons and matching-rule explanations beneath. Row dividers maintain the open-list structure. Outcome text remains present alongside color.

### Result Mosaic

Result counts occupy flush rectangular color cells separated by a small gutter (2px), without an outside card border or shadow. The last-run treatment gives Moved a full-width green cell, then four subordinate cells for Duplicates, Renamed, Skipped and Failed. General summaries use five columns. Last-run cells temporarily use two columns at intermediate widths, then four below the full-width moved cell on small phones.

Counts use tabular serif numerals and smaller sans-serif labels. Render real run data or explicitly labeled demo data. Empty states use explanatory copy and a divider rather than invented counts.

### Switches and Settings Rows

Switches pair a restrained outlined track with a circular knob. Checked state fills the track with Ink and shifts the Paper knob horizontally (18px). The track is compact; keep its adjacent text label readable and clickable. Each row groups its label and explanatory text opposite the switch, with a subordinate bottom rule.

### Disclosures and Save Actions

Advanced settings, category editors and history runs use native disclosures within ruled sections. They expand in place. Save/Discard is a ruled action row, not an overlay; unsaved-state text and disabled states communicate whether changes can be applied.

### Artwork and Motion

The original globe/file artwork is inline SVG with thin outlines, pastel fills and a serif note. It is decorative and hidden from assistive technology; action icons are also hidden when their text supplies the accessible label. It scales within its allocated hero space rather than becoming a background image behind text.

The only authored entrance is a short artwork reveal (650ms), combining a small upward arrival and clip reveal. State changes use the shared easing curve (`cubic-bezier(0.23, 1, 0.32, 1)`) over 180ms. Reduced-motion preference removes animations, transitions and smooth scrolling. Do not introduce perpetual artwork motion.

## Do's and Don'ts

### Do:

- **Do** preserve the single light theme and locally bundled font pairing.
- **Do** align open sections to the shell and separate them with space and fine rules.
- **Do** keep outcome labels, helper text, long paths and keyboard focus visible.
- **Do** use the established SVG action arrow consistently when an action needs an icon.
- **Do** keep Save/Discard in normal flow and keep artwork clear of headings at narrow widths.
- **Do** respect reduced motion and label simulated data explicitly.

### Don't:

- **Don't** add rounded cards, enclosing dashboard panels or boxed fieldsets.
- **Don't** add automatic dark-mode switching or a theme selector.
- **Don't** replace the original line artwork with generic decorative icons or unrelated stock imagery.
- **Don't** use shadows to separate ordinary sections or animate the artwork continuously.
- **Don't** invent file statistics or imply that browser demonstration proves Android-manager support.

Not canonized: unused declared Surface and Pink variables are excluded from the recorded palette. The textual destination marker (`↳`) in file paths describes a path relationship; it is not an action icon or a pending runtime defect.
