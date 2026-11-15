import { createBridge } from "./bridge.mjs";
import { createDemo } from "./demo.mjs";
import {
  validateSettings,
  settingsFromForm,
  overlapWarnings,
  reorder,
  filterPreview,
  PRESETS,
  newestRuns,
} from "./model.mjs";

const $ = (id) => document.getElementById(id);
const isDemo = new URLSearchParams(location.search).get("demo") === "1";
const api = isDemo ? createDemo() : createBridge(window);
const state = {
  snapshot: null,
  draft: null,
  dirty: false,
  pending: false,
  fault: false,
  preview: null,
  previewAt: null,
  exportText: "",
  page: "home",
  disposed: false,
  refreshing: false,
  historyKey: "",
};
let pollTimer;
let snapshotRequest;
let noticeTimer;
let confirmResolve;
const count = (value) => (Number.isInteger(value) ? value : 0);
const date = (value) =>
  typeof value === "number" && value > 0
    ? new Intl.DateTimeFormat(undefined, {
        day: "2-digit",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }).format(value * 1000)
    : "Not yet";
const basename = (path) =>
  typeof path === "string" ? path.split("/").pop() : "";
const categoryName = (id) =>
  state.snapshot?.settings.categories.find((category) => category.id === id)
    ?.name ||
  id ||
  "Unclassified";
const ruleName = (id) =>
  state.snapshot?.settings.rules.find((rule) => rule.id === id)?.name || id;
const runLabel = (run) =>
  ({
    complete: "Complete",
    completed: "Complete",
    complete_with_errors: "Completed with errors",
    completed_with_errors: "Completed with errors",
    failed: "Failed",
    stopped: "Stopped",
    running: "Sorting",
    interrupted: "Interrupted",
  })[run?.state] ||
  run?.state ||
  "No runs yet";
function element(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (key.startsWith("on")) node.addEventListener(key.slice(2), value);
    else if (key === "class") node.className = value;
    else if (key === "text") node.textContent = value;
    else if (
      key === "checked" ||
      key === "disabled" ||
      key === "hidden" ||
      key === "open"
    )
      node[key] = value;
    else if (key === "value") node.value = value;
    else node.setAttribute(key, value);
  }
  for (const child of [children].flat(Infinity))
    if (child !== null && child !== undefined)
      node.append(
        child instanceof Node ? child : document.createTextNode(child),
      );
  return node;
}
function button(text, handler, attrs = {}) {
  return element(
    "button",
    { type: "button", "data-write": "", onclick: handler, ...attrs },
    text,
  );
}
function notify(message, type = "success") {
  clearTimeout(noticeTimer);
  $("notice").textContent = message;
  $("notice").className = `notice ${type}`;
  $("notice").hidden = false;
  $("notice").setAttribute("role", type === "error" ? "alert" : "status");
  if (type !== "error")
    noticeTimer = setTimeout(() => {
      $("notice").hidden = true;
    }, 10000);
}
function warnings(target, messages) {
  target.replaceChildren();
  if (!messages?.length) return;
  target.append(
    element(
      "div",
      { class: "warning-list" },
      messages.map((message) => element("p", {}, message)),
    ),
  );
}
function summaryView(summary) {
  return element(
    "div",
    { class: "metrics", "aria-label": "File counts" },
    [
      ["moved", "Moved"],
      ["duplicates", "Duplicates"],
      ["renamed", "Renamed"],
      ["skipped", "Skipped"],
      ["failed", "Failed"],
    ].map(([key, label]) =>
      element("div", { class: "metric" }, [
        element("strong", {}, count(summary?.[key])),
        element("span", {}, label),
      ]),
    ),
  );
}
function confirmAction(title, description, accept = "Continue") {
  if (confirmResolve) return Promise.resolve(false);
  $("confirm-title").textContent = title;
  $("confirm-description").textContent = description;
  $("confirm-accept").textContent = accept;
  $("confirm-dialog").returnValue = "";
  $("confirm-dialog").showModal();
  return new Promise((resolve) => {
    confirmResolve = resolve;
  });
}
$("confirm-dialog").addEventListener("close", () => {
  const resolve = confirmResolve;
  confirmResolve = null;
  resolve?.($("confirm-dialog").returnValue === "confirm");
});
function syncControls() {
  const connected = api.supported && state.snapshot && !state.fault;
  const running = Boolean(state.snapshot?.status.running);
  document.body.classList.toggle("busy", state.pending);
  document.querySelectorAll("[data-write]").forEach((control) => {
    control.disabled =
      !connected ||
      state.pending ||
      (running && !["stop-run", "home-auto"].includes(control.id));
  });
  document.querySelectorAll("[data-settings]").forEach((fieldset) => {
    fieldset.disabled = !connected || state.pending || running;
  });
  $("stop-run").disabled =
    !connected ||
    state.pending ||
    !running ||
    Boolean(state.snapshot?.status.stopping);
  $("save-settings").disabled =
    !connected || state.pending || running || !state.dirty;
  $("discard-settings").disabled =
    !connected || state.pending || running || !state.dirty;
  $("refresh-home").disabled =
    state.pending || state.refreshing || !api.supported;
  $("dirty-label").textContent = state.dirty
    ? "Unsaved changes"
    : "No unsaved changes";
  $("download-export").disabled = !state.exportText;
  document
    .querySelectorAll("[data-rule-boundary=true], [data-no-undo=true]")
    .forEach((control) => {
      control.disabled = true;
    });
}
function setDirty() {
  state.dirty = true;
  syncControls();
}
function intervalLabel(seconds) {
  if (seconds % 3600 === 0)
    return `Every ${seconds / 3600} ${seconds === 3600 ? "hour" : "hours"}`;
  if (seconds % 60 === 0) return `Every ${seconds / 60} minutes`;
  return `Every ${seconds} seconds`;
}
function renderHome() {
  if (!state.snapshot) return;
  const { status, settings } = state.snapshot;
  $("version").textContent = `v${state.snapshot.version}`;
  const active = status.running;
  $("run-state").textContent = state.fault
    ? "Status unavailable"
    : status.stopping
      ? "Stopping"
      : active
        ? status.operation === "undo"
          ? "Restoring"
          : "Sorting"
        : status.error
          ? "Needs attention"
          : "Ready";
  $("run-state").className =
    `status-pill${state.fault || status.error ? " offline" : active ? " busy" : ""}`;
  $("service-state").textContent = status.serviceRunning
    ? "Service running"
    : "Service not running";
  $("home-auto").checked = settings.autoSort;
  $("home-auto-description").textContent = settings.autoSort
    ? intervalLabel(settings.intervalSeconds)
    : `Off · ${intervalLabel(settings.intervalSeconds).toLowerCase()} when enabled`;
  $("home-delay").textContent = settings.delayEnabled ? "1 day" : "Off";
  $("next-run").textContent = settings.autoSort
    ? status.nextRun
      ? date(status.nextRun)
      : status.serviceRunning
        ? "Waiting for service"
        : "Service unavailable"
    : "Automatic sorting off";
  $("sort-now").hidden = active;
  $("stop-run").hidden = !active;
  $("current-file").hidden = !status.currentFile;
  $("current-file").textContent = status.currentFile
    ? `${status.operation === "undo" ? "Restoring" : "Current file"}: ${status.currentFile}`
    : "";
  $("run-help").textContent = status.stopping
    ? "Finishing the current file before stopping. Completed moves remain in History."
    : active
      ? "Stop finishes the current file. Future automatic checks stay enabled if configured."
      : "Manual sorting follows your delay, ignore rules and Undo protection.";
  const run = status.lastRun;
  $("last-run").replaceChildren();
  if (run) {
    $("last-run").append(
      element("h3", {}, runLabel(run)),
      element(
        "time",
        {
          datetime: new Date(
            (run.finishedAt || run.startedAt) * 1000,
          ).toISOString(),
        },
        date(run.finishedAt || run.startedAt),
      ),
      summaryView(run.summary),
    );
    if (run.summary?.duplicates || run.summary?.renamed)
      $("last-run").append(
        element(
          "p",
          { class: "run-caption" },
          "Duplicates and renamed files are included in Moved.",
        ),
      );
    if (run.error)
      $("last-run").append(element("p", { class: "reason" }, run.error));
    if (run.undo) $("last-run").append(undoSummary(run.undo));
  } else
    $("last-run").append(
      element(
        "p",
        { class: "empty-state" },
        "No sorting runs yet. Preview your files before the first sort.",
      ),
    );
  warnings($("home-warnings"), [
    ...(state.snapshot.warnings || []),
    ...(status.error ? [status.error] : []),
  ]);
}
function undoSummary(undo) {
  return element(
    "p",
    { class: "reason" },
    `Last Undo: ${count(undo.restored)} restored · ${count(undo.conflicted)} conflicted · ${count(undo.failed)} failed`,
  );
}
function fileView(item, { historyRun = null } = {}) {
  const row = element("article", { class: "file-row" });
  const outcome = item.undone
    ? "Restored"
    : {
        move: item.renamed ? "Renamed" : "Move",
        duplicate: "Duplicate",
        skip: "Skipped",
        error: "Error",
      }[item.outcome] || item.outcome;
  row.append(
    element("div", { class: "file-title-row" }, [
      element("h3", { class: "file-title" }, basename(item.source)),
      element("span", { class: `outcome ${item.outcome}` }, outcome),
    ]),
  );
  row.append(element("p", { class: "file-path" }, item.source));
  if (item.destination)
    row.append(
      element("p", { class: "file-path" }, [
        element("span", { class: "arrow", "aria-hidden": "true" }, "↳"),
        item.destination,
      ]),
    );
  if (item.reason) row.append(element("p", { class: "reason" }, item.reason));
  if (item.duplicateOf)
    row.append(
      element("p", { class: "file-path" }, `Identical to: ${item.duplicateOf}`),
    );
  if (item.category)
    row.append(
      element(
        "p",
        { class: "hint" },
        `Category: ${categoryName(item.category)}`,
      ),
    );
  if (item.matchingRules?.length)
    row.append(
      element(
        "p",
        { class: "hint" },
        `Matching rules: ${item.matchingRules.map(ruleName).join(", ")}. Winner: ${ruleName(item.winner) || "None"}.`,
      ),
    );
  else if (item.winner)
    row.append(
      element(
        "p",
        { class: "hint" },
        item.winner.startsWith("extension:")
          ? `Matched extension: .${item.winner.slice(10)}`
          : item.winner === "fallback"
            ? "Matched unknown-file fallback"
            : `Matched: ${item.winner}`,
      ),
    );
  if (
    historyRun &&
    ["move", "duplicate"].includes(item.outcome) &&
    !item.undone
  )
    row.append(
      element("div", { class: "actions" }, [
        button("Undo this file", () => undo(historyRun.id, item.id)),
      ]),
    );
  return row;
}
function renderPreview() {
  const data = state.preview;
  if (!data) return;
  $("preview-time").textContent =
    `Scanned ${date(state.previewAt)}${state.dirty ? " · using saved settings" : ""}`;
  $("preview-summary").replaceChildren(summaryView(data.summary));
  warnings($("preview-warnings"), data.warnings || []);
  const items = filterPreview(
    data.items || [],
    $("preview-filter").value,
    $("preview-search").value,
  );
  $("preview-list").replaceChildren(...items.map((item) => fileView(item)));
  if (!items.length)
    $("preview-list").append(
      element(
        "p",
        { class: "empty-state" },
        data.items?.length
          ? "No files match these filters."
          : "No direct files to preview in the source folder.",
      ),
    );
}
function renderHistory() {
  if (!state.snapshot) return;
  const { history, holds, settings } = state.snapshot;
  const historyKey = JSON.stringify([
    history,
    holds,
    settings.historyDays,
    settings.categories,
  ]);
  if (historyKey === state.historyKey) return;
  state.historyKey = historyKey;
  $("retention-label").textContent =
    `History is kept for up to ${settings.historyDays} days or the latest 200 runs, whichever limit comes first.`;
  const openRuns = new Set(
    [...$("history-list").querySelectorAll("details[open]")].map(
      (node) => node.dataset.runId,
    ),
  );
  $("history-list").replaceChildren();
  for (const run of newestRuns(history || [])) {
    const details = element("details", {
      class: "history-run",
      "data-run-id": run.id,
      open: openRuns.has(run.id),
    });
    details.append(
      element(
        "summary",
        {},
        element("div", {}, [
          element("h3", {}, runLabel(run)),
          element("time", {}, date(run.finishedAt || run.startedAt)),
        ]),
      ),
    );
    details.append(summaryView(run.summary));
    if (run.error) details.append(element("p", { class: "reason" }, run.error));
    if (run.undo) details.append(undoSummary(run.undo));
    const undoable = (run.entries || []).filter(
      (entry) => ["move", "duplicate"].includes(entry.outcome) && !entry.undone,
    );
    details.append(
      element("div", { class: "run-actions" }, [
        element(
          "p",
          {},
          `${undoable.length} ${undoable.length === 1 ? "file" : "files"} available for Undo. Restores run in reverse move order.`,
        ),
        button("Undo entire run", () => undo(run.id), {
          "data-run-undo": "",
          "data-no-undo": undoable.length ? "false" : "true",
          disabled: !undoable.length,
        }),
      ]),
    );
    details.append(
      ...(run.entries || []).map((entry) =>
        fileView(entry, { historyRun: run }),
      ),
    );
    $("history-list").append(details);
  }
  if (!history?.length)
    $("history-list").append(
      element(
        "p",
        { class: "empty-state" },
        "No move history yet. New sorting runs will appear here.",
      ),
    );
  $("holds-count").textContent =
    `${holds?.length || 0} held ${holds?.length === 1 ? "file" : "files"}`;
  $("holds-list").replaceChildren(
    ...(holds || []).map((hold) =>
      element("div", { class: "hold-row" }, [
        element("p", { class: "file-path" }, hold.path),
        button("Allow sorting again", async () => {
          if (
            await confirmAction(
              "Allow this file to sort again?",
              `${hold.path}\n\nThe file will follow your saved sorting rules at the next manual or automatic run.`,
              "Release protection",
            )
          )
            await perform(
              { command: "holds.release", path: hold.path },
              "Undo protection released.",
            );
        }),
      ]),
    ),
  );
  if (!holds?.length)
    $("holds-list").append(
      element("p", { class: "empty-state" }, "No files are protected by Undo."),
    );
  syncControls();
  document.querySelectorAll("[data-run-undo]").forEach((control) => {
    const details = control.closest("[data-run-id]");
    const run = (history || []).find((run) => run.id === details.dataset.runId);
    if (
      !(run?.entries || []).some(
        (entry) =>
          ["move", "duplicate"].includes(entry.outcome) && !entry.undone,
      )
    )
      control.disabled = true;
  });
}
async function refresh(options = {}) {
  if (state.refreshing) return snapshotRequest;
  snapshotRequest = readSnapshot(options);
  return snapshotRequest;
}
async function readSnapshot({ quiet = false } = {}) {
  if (state.refreshing || state.disposed || !api.supported) return false;
  state.refreshing = true;
  syncControls();
  try {
    const snapshot = await api.request({ command: "snapshot" });
    if (
      !snapshot?.settings ||
      !snapshot?.status ||
      !Array.isArray(snapshot.history) ||
      !Array.isArray(snapshot.holds)
    )
      throw new Error(
        "Device snapshot is incomplete. Update the installed module.",
      );
    state.snapshot = snapshot;
    state.fault = false;
    $("connection").className = isDemo
      ? "connection demo-status"
      : "connection";
    $("connection").textContent = isDemo
      ? "Demo · simulated service and file activity"
      : "";
    if (
      !state.draft ||
      (!state.dirty &&
        JSON.stringify(state.draft) !== JSON.stringify(snapshot.settings))
    ) {
      state.draft = structuredClone(snapshot.settings);
      renderSettings();
    }
    renderHome();
    renderHistory();
    if (!quiet && state.page === "home" && state.snapshot.status.error)
      notify(state.snapshot.status.error, "error");
    return true;
  } catch (error) {
    state.fault = true;
    $("connection").className = "connection unsupported";
    $("connection").textContent =
      "Device status unavailable. Displayed data may be stale. Refresh to reconnect.";
    renderHome();
    notify(error.message, "error");
    return false;
  } finally {
    state.refreshing = false;
    syncControls();
  }
}
async function perform(request, message) {
  if (state.pending || !api.supported || !state.snapshot || state.fault)
    return false;
  state.pending = true;
  syncControls();
  try {
    await api.request(request);
    if (state.refreshing) await snapshotRequest;
    const current = await refresh({ quiet: true });
    if (current && message) notify(`${isDemo ? "Demo: " : ""}${message}`);
    return current;
  } catch (error) {
    notify(error.message, "error");
    return false;
  } finally {
    state.pending = false;
    syncControls();
  }
}
async function undo(runId, entryId) {
  const confirmed = await confirmAction(
    entryId ? "Restore this file?" : "Restore this entire run?",
    "Only unchanged moved files can be restored. Occupied original paths are left intact. Successfully restored files stay protected from sorting until you release them.",
    entryId ? "Undo file" : "Undo run",
  );
  if (!confirmed) return;
  await perform(
    { command: "undo", runId, ...(entryId ? { entryId } : {}) },
    "Undo requested. Check History for restored, conflicted and failed counts.",
  );
}
async function scanPreview() {
  if (
    state.pending ||
    !state.snapshot ||
    state.snapshot.status.running ||
    state.fault
  )
    return;
  state.pending = true;
  syncControls();
  $("refresh-preview").textContent = "Scanning…";
  try {
    const result = await api.request({ command: "preview" });
    if (!result || !Array.isArray(result.items))
      throw new Error("Device returned an invalid preview.");
    state.preview = result;
    state.previewAt = Math.floor(Date.now() / 1000);
    renderPreview();
    notify(`${isDemo ? "Demo: " : ""}Preview ready. No files were moved.`);
  } catch (error) {
    notify(error.message, "error");
  } finally {
    state.pending = false;
    $("refresh-preview").textContent = "Scan preview ↗";
    syncControls();
  }
}
function makeField(
  label,
  value,
  oninput,
  { hint = "", type = "text", rows = 3 } = {},
) {
  const id = `field-${++fieldSequence}`;
  const input = element(type === "textarea" ? "textarea" : "input", {
    id,
    "aria-label": label,
    ...(hint ? { "aria-describedby": `${id}-hint` } : {}),
    ...(type === "textarea" ? { rows } : { type }),
    value,
    spellcheck: "false",
    oninput: (event) => oninput(event.target.value),
  });
  return element("label", { class: "field", for: id }, [
    label,
    input,
    ...(hint
      ? [element("span", { class: "hint", id: `${id}-hint` }, hint)]
      : []),
  ]);
}
let fieldSequence = 0;
let expandedCategory = "";
function renderCategories() {
  const draft = state.draft;
  if (!draft) return;
  const expanded = new Set(
    [...$("category-editors").querySelectorAll("details[open]")].map(
      (node) => node.dataset.categoryId,
    ),
  );
  if (expandedCategory) expanded.add(expandedCategory);
  expandedCategory = "";
  $("category-toggles").replaceChildren();
  $("category-editors").replaceChildren();
  draft.categories.forEach((category, index) => {
    const toggle = element("input", {
      type: "checkbox",
      class: "switch",
      id: `category-toggle-${index}`,
      checked: category.enabled,
      onchange: (event) => {
        category.enabled = event.target.checked;
        setDirty();
        renderCategories();
      },
    });
    $("category-toggles").append(
      element("div", { class: "category-toggle" }, [
        element("label", { for: `category-toggle-${index}` }, [
          category.name,
          element(
            "small",
            {},
            category.extensions.length
              ? `${category.extensions.slice(0, 7).join(", ")}${category.extensions.length > 7 ? "…" : ""}`
              : "Unknown file types",
          ),
        ]),
        toggle,
      ]),
    );
    const editor = element("details", {
      class: "editor-item category-editor",
      "data-category-id": category.id,
      open: expanded.has(category.id),
    });
    editor.append(
      element("summary", {}, [
        element("strong", { class: "editor-name" }, category.name),
        element("span", { class: "muted small" }, "Edit"),
      ]),
    );
    const removeControl =
      category.id !== "others"
        ? button(
            "Remove category",
            async () => {
              const references = draft.rules.filter(
                (rule) => rule.category === category.id,
              );
              if (references.length) {
                notify(
                  `Remove or reassign rules using ${category.name} before removing this category.`,
                  "error",
                );
                return;
              }
              if (
                await confirmAction(
                  `Remove ${category.name}?`,
                  "Existing files remain in their folders. New files will use the remaining rules and unknown-file behavior.",
                  "Remove category",
                )
              ) {
                draft.categories.splice(index, 1);
                setDirty();
                renderCategories();
                renderRules();
              }
            },
            {
              class: "text-button danger-text",
              "aria-label": `Remove ${category.name} category`,
            },
          )
        : null;
    editor.append(
      makeField(
        "Category name",
        category.name,
        (value) => {
          category.name = value;
          setDirty();
          const label = $(`category-toggle-${index}`).previousElementSibling;
          label.firstChild.textContent = value || "Unnamed category";
          editor.querySelector(".editor-name").textContent =
            value || "Unnamed category";
          renderRules();
        },
        {
          hint: "Renaming a label does not move or rename its destination folder.",
        },
      ),
    );
    editor.append(
      makeField(
        "Extensions",
        category.extensions.join(", "),
        (value) => {
          category.extensions = value
            .split(/[\s,]+/)
            .filter(Boolean)
            .map((extension) => extension.toLowerCase());
          setDirty();
        },
        {
          hint: "Comma-separated, without leading dots. Compound suffixes such as tar.gz are supported. More specific suffixes win.",
        },
      ),
    );
    editor.append(
      makeField("Destination folder", category.destination, (value) => {
        category.destination = value;
        setDirty();
      }),
    );
    if (removeControl) editor.append(removeControl);
    $("category-editors").append(editor);
  });
  $("preset").replaceChildren(
    element("option", { value: "" }, "Choose optional preset"),
    ...PRESETS.map((preset) =>
      element(
        "option",
        { value: preset.id },
        `${draft.categories.some((category) => category.id === preset.id) ? "Enable" : "Add"} ${preset.name}`,
      ),
    ),
  );
  syncControls();
}
function renderRuleWarnings() {
  warnings($("rule-warnings"), overlapWarnings(state.draft?.rules || []));
}
function renderRules() {
  const draft = state.draft;
  if (!draft) return;
  $("rule-editors").replaceChildren();
  draft.rules.forEach((rule, index) => {
    const item = element("div", { class: "editor-item" });
    item.append(
      element("div", { class: "editor-topline" }, [
        element("label", { class: "category-state" }, [
          element("input", {
            type: "checkbox",
            checked: rule.enabled,
            onchange: (event) => {
              rule.enabled = event.target.checked;
              setDirty();
              renderRuleWarnings();
            },
          }),
          `Rule ${index + 1}`,
        ]),
        element("div", { class: "reorder-controls" }, [
          button(
            "↑",
            () => {
              draft.rules = reorder(draft.rules, index, -1);
              setDirty();
              renderRules();
            },
            {
              "aria-label": `Move ${rule.name} up`,
              "data-rule-boundary": index === 0 ? "true" : "false",
            },
          ),
          button(
            "↓",
            () => {
              draft.rules = reorder(draft.rules, index, 1);
              setDirty();
              renderRules();
            },
            {
              "aria-label": `Move ${rule.name} down`,
              "data-rule-boundary":
                index === draft.rules.length - 1 ? "true" : "false",
            },
          ),
          button(
            "Remove",
            () => {
              draft.rules.splice(index, 1);
              setDirty();
              renderRules();
            },
            {
              class: "text-button danger-text",
              "aria-label": `Remove ${rule.name} rule`,
            },
          ),
        ]),
      ]),
    );
    item.append(
      makeField("Rule name", rule.name, (value) => {
        rule.name = value;
        setDirty();
        renderRuleWarnings();
      }),
    );
    item.append(
      element("div", { class: "editor-grid" }, [
        makeField(
          "Filename pattern",
          rule.pattern,
          (value) => {
            rule.pattern = value;
            setDirty();
            renderRuleWarnings();
          },
          { hint: "Example: Invoice-*.pdf" },
        ),
        element("label", { class: "field" }, [
          "Destination category",
          element(
            "select",
            {
              onchange: (event) => {
                rule.category = event.target.value;
                setDirty();
              },
            },
            draft.categories.map((category) =>
              element(
                "option",
                {
                  value: category.id,
                  ...(category.id === rule.category ? { selected: "" } : {}),
                },
                category.name,
              ),
            ),
          ),
        ]),
      ]),
    );
    $("rule-editors").append(item);
  });
  if (!draft.rules.length)
    $("rule-editors").append(
      element(
        "p",
        { class: "empty-state" },
        "No custom rules. Files follow extension mappings.",
      ),
    );
  renderRuleWarnings();
  syncControls();
  document.querySelectorAll("[data-rule-boundary=true]").forEach((control) => {
    control.disabled = true;
  });
}
function renderSettings() {
  const draft = state.draft;
  if (!draft) return;
  $("auto-sort").checked = draft.autoSort;
  $("delay-enabled").checked = draft.delayEnabled;
  $("interval").value = draft.intervalSeconds;
  $("unknown-files").value = draft.unknownFiles;
  $("ignore").value = draft.ignore.join("\n");
  $("source-path").value = draft.source;
  $("destination-path").value = draft.destination;
  $("history-days").value = draft.historyDays;
  renderCategories();
  renderRules();
  syncControls();
}
function resetDraft() {
  state.draft = structuredClone(state.snapshot.settings);
  state.dirty = false;
  renderSettings();
}
function newId(prefix) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}
function addCategory(preset = null) {
  if (!state.draft) return;
  if (state.draft.categories.length >= 100) {
    notify("Settings allow up to 100 categories.", "error");
    return;
  }
  const existing =
    preset &&
    state.draft.categories.find((category) => category.id === preset.id);
  if (existing) existing.enabled = true;
  else {
    const name =
      preset?.name || `New category ${state.draft.categories.length + 1}`;
    const id = preset?.id || newId("category");
    expandedCategory = id;
    state.draft.categories.push({
      id,
      name,
      enabled: true,
      extensions: [...(preset?.extensions || [])],
      destination: `${state.draft.destination.replace(/\/$/, "")}/${name}`,
    });
  }
  setDirty();
  renderCategories();
  renderRules();
}
async function saveSettings(event) {
  event.preventDefault();
  if (!state.draft || state.pending) return;
  try {
    const candidate = settingsFromForm(state.draft, {
      intervalSeconds: $("interval").value,
      historyDays: $("history-days").value,
    });
    if (
      candidate.historyDays < state.snapshot.settings.historyDays &&
      !(await confirmAction(
        "Keep less history?",
        `Reducing retention from ${state.snapshot.settings.historyDays} to ${candidate.historyDays} days removes older Undo records when history is pruned. Organized files and Undo protection remain.`,
        "Save shorter retention",
      ))
    )
      return;
    if (
      candidate.autoSort &&
      !state.snapshot.settings.autoSort &&
      !(await confirmAction(
        "Enable automatic sorting?",
        `${intervalLabel(candidate.intervalSeconds)}. ${candidate.delayEnabled ? "Files are kept for 1 day before becoming eligible." : "The one-day delay is off."}\n\nPreview uses saved settings. Save with automatic sorting off first if you want to preview new rules.`,
        "Enable & save",
      ))
    )
      return;
    const saved = await perform(
      { command: "settings.save", settings: candidate },
      "Settings saved. The scheduler will use your updated settings.",
    );
    if (saved) {
      state.dirty = false;
      state.draft = structuredClone(state.snapshot.settings);
      renderSettings();
      invalidatePreview();
    }
  } catch (error) {
    notify(error.message, "error");
  }
}
function parseImport() {
  const raw = $("settings-json").value;
  if (!raw.trim())
    throw new Error("Paste a settings JSON document before importing.");
  if (new TextEncoder().encode(raw).length > 64 * 1024)
    throw new Error("Settings JSON exceeds the 64 KiB request limit.");
  let settings;
  try {
    settings = JSON.parse(raw);
  } catch (error) {
    throw new Error(`Settings JSON could not be parsed: ${error.message}`, {
      cause: error,
    });
  }
  if (settings?.settings && settings.schema === undefined)
    settings = settings.settings;
  validateSettings(settings);
  return settings;
}
function invalidatePreview() {
  state.preview = null;
  $("preview-list").replaceChildren(
    element(
      "p",
      { class: "empty-state" },
      "Settings changed. Scan a new preview to see the updated plan.",
    ),
  );
  $("preview-summary").replaceChildren();
  $("preview-warnings").replaceChildren();
  $("preview-time").textContent = "Settings changed · scan again";
}
function importSummary(settings) {
  return `${settings.categories.length} categories · ${settings.rules.length} rules · ${settings.ignore.length} ignore patterns. ${settings.autoSort ? "Automatic sorting on" : "Automatic sorting off"}. Source: ${settings.source}. Root: ${settings.destination}. Retention: up to ${settings.historyDays} days or the latest 200 runs, whichever limit comes first.`;
}
function route() {
  const requested = location.hash.slice(1) || "home";
  state.page = ["home", "preview", "history", "settings"].includes(requested)
    ? requested
    : "home";
  document.querySelectorAll("[id^=page-]").forEach((section) => {
    section.hidden = section.id !== `page-${state.page}`;
  });
  document.querySelectorAll("[data-page]").forEach((link) => {
    if (link.dataset.page === state.page)
      link.setAttribute("aria-current", "page");
    else link.removeAttribute("aria-current");
  });
  document.title = `${state.page === "home" ? "Sortify" : `${state.page[0].toUpperCase()}${state.page.slice(1)} · Sortify`}${isDemo ? " · Demo" : ""}`;
  if (state.page === "preview") renderPreview();
}
$("home-auto").addEventListener("change", async (event) => {
  const enabled = event.target.checked;
  event.target.checked = state.snapshot.settings.autoSort;
  if (
    enabled &&
    !(await confirmAction(
      "Enable automatic sorting?",
      `${intervalLabel(state.snapshot.settings.intervalSeconds)}. Preview eligible files first if you have not reviewed them. Manual sorting remains available when this is off.`,
      "Enable automatic sorting",
    ))
  )
    return;
  const settings = { ...state.snapshot.settings, autoSort: enabled };
  if (
    await perform(
      { command: "settings.save", settings },
      enabled
        ? "Automatic sorting enabled."
        : "Automatic sorting disabled. Manual sorting is still available.",
    )
  ) {
    if (state.dirty) {
      state.draft.autoSort = enabled;
      $("auto-sort").checked = enabled;
    } else resetDraft();
  }
});
$("sort-now").addEventListener("click", () =>
  perform(
    { command: "start" },
    "Sorting requested. Live status will show the actual result.",
  ),
);
$("stop-run").addEventListener("click", () =>
  perform(
    { command: "stop" },
    "Stop requested. The current file will finish before the run stops.",
  ),
);
$("open-preview").addEventListener("click", () => {
  location.hash = "preview";
  scanPreview();
});
$("refresh-home").addEventListener("click", () => refresh());
$("refresh-preview").addEventListener("click", scanPreview);
$("preview-filter").addEventListener("change", renderPreview);
$("preview-search").addEventListener("input", renderPreview);
$("clear-history").addEventListener("click", async () => {
  if (
    await confirmAction(
      "Clear move history?",
      "Cleared records permanently lose Undo. Organized files and Undo protection are kept. Unfinished operations are retained until recovery is complete.",
      "Clear history",
    )
  )
    await perform(
      { command: "history.clear" },
      "History cleared. Organized files and Undo protection were kept.",
    );
});
$("auto-sort").addEventListener("change", (event) => {
  state.draft.autoSort = event.target.checked;
  setDirty();
});
$("delay-enabled").addEventListener("change", (event) => {
  state.draft.delayEnabled = event.target.checked;
  setDirty();
});
$("unknown-files").addEventListener("change", (event) => {
  state.draft.unknownFiles = event.target.value;
  setDirty();
});
$("interval").addEventListener("input", setDirty);
$("history-days").addEventListener("input", setDirty);
$("ignore").addEventListener("input", (event) => {
  state.draft.ignore = event.target.value
    .split(/\r?\n/)
    .filter((pattern) => pattern.trim());
  setDirty();
});
$("source-path").addEventListener("input", (event) => {
  state.draft.source = event.target.value;
  setDirty();
});
$("destination-path").addEventListener("change", (event) => {
  const old = state.draft.destination.replace(/\/$/, "");
  const value = event.target.value.replace(/\/$/, "");
  state.draft.categories.forEach((category) => {
    if (
      category.destination === old ||
      category.destination.startsWith(`${old}/`)
    )
      category.destination = value + category.destination.slice(old.length);
  });
  state.draft.destination = value;
  setDirty();
  renderCategories();
});
$("destination-path").addEventListener("input", setDirty);
$("settings-form").addEventListener("submit", saveSettings);
$("discard-settings").addEventListener("click", async () => {
  if (
    await confirmAction(
      "Discard unsaved changes?",
      "Your saved settings remain active.",
      "Discard changes",
    )
  )
    resetDraft();
});
$("add-category").addEventListener("click", () => addCategory());
$("add-preset").addEventListener("click", () => {
  const preset = PRESETS.find((preset) => preset.id === $("preset").value);
  if (!preset) {
    notify("Choose an optional category preset first.", "info");
    return;
  }
  addCategory(preset);
});
$("add-rule").addEventListener("click", () => {
  if (state.draft.rules.length >= 200) {
    notify("Settings allow up to 200 filename rules.", "error");
    return;
  }
  state.draft.rules.push({
    id: newId("rule"),
    name: `Rule ${state.draft.rules.length + 1}`,
    enabled: true,
    pattern: "*",
    category: state.draft.categories[0].id,
  });
  setDirty();
  renderRules();
});
$("export-settings").addEventListener("click", async () => {
  if (state.pending) return;
  state.pending = true;
  syncControls();
  try {
    const settings = await api.request({ command: "settings.export" });
    validateSettings(settings);
    state.exportText = `${JSON.stringify(settings, null, 2)}\n`;
    $("settings-json").value = state.exportText;
    $("download-export").hidden = false;
    $("import-summary").textContent =
      "Saved settings exported. Select and copy the JSON or download a local file.";
    notify(
      `${isDemo ? "Demo: " : ""}Saved settings exported. Unsaved edits are excluded.`,
    );
  } catch (error) {
    notify(error.message, "error");
  } finally {
    state.pending = false;
    syncControls();
  }
});
$("download-export").addEventListener("click", () => {
  if (!state.exportText) return;
  const url = URL.createObjectURL(
    new Blob([state.exportText], { type: "application/json" }),
  );
  const link = element("a", { href: url, download: "sortify-settings.json" });
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
});
$("settings-json").addEventListener("input", () => {
  state.exportText = "";
  $("download-export").hidden = true;
  if (!$("settings-json").value.trim()) {
    $("import-summary").textContent = "";
    return;
  }
  try {
    $("import-summary").textContent = importSummary(parseImport());
  } catch (error) {
    $("import-summary").textContent = error.message;
  }
});
$("import-settings").addEventListener("click", async () => {
  try {
    const settings = parseImport();
    if (
      !(await confirmAction(
        "Import these settings?",
        `${importSummary(settings)}\n\nThis replaces saved settings and unsaved edits. Shorter retention removes older Undo records when pruned. Device folder access will be checked before applying.`,
        "Import settings",
      ))
    )
      return;
    if (
      await perform(
        { command: "settings.import", settings },
        "Settings imported and folder access checked. Review Preview before sorting.",
      )
    ) {
      resetDraft();
      invalidatePreview();
    }
  } catch (error) {
    notify(error.message, "error");
  }
});
window.addEventListener("hashchange", route);
function schedulePoll() {
  clearTimeout(pollTimer);
  if (state.disposed || document.hidden || !api.supported) return;
  pollTimer = setTimeout(
    async () => {
      if (!state.pending) await refresh({ quiet: true });
      schedulePoll();
    },
    state.snapshot?.status.running ? 2000 : 5000,
  );
}
function visibilityChanged() {
  clearTimeout(pollTimer);
  if (!document.hidden) {
    refresh({ quiet: true });
    schedulePoll();
  }
}
document.addEventListener("visibilitychange", visibilityChanged);
window.addEventListener("pagehide", () => {
  state.disposed = true;
  clearTimeout(pollTimer);
  clearTimeout(noticeTimer);
});
window.addEventListener("pageshow", () => {
  if (state.disposed) {
    state.disposed = false;
    refresh({ quiet: true });
    schedulePoll();
  }
});
route();
if (isDemo) $("demo-banner").hidden = false;
if (!api.supported) {
  $("connection").className = "connection unsupported";
  $("connection").textContent =
    "Device access unavailable. Open Sortify in a KernelSU manager with the supported WebUI callback bridge. Controls are disabled here.";
  $("run-state").textContent = "Unsupported environment";
  $("run-state").className = "status-pill offline";
  $("service-state").textContent = "No device connection";
  syncControls();
} else {
  refresh().then(schedulePoll);
}
