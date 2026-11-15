const isRecord = (value) =>
  value !== null && typeof value === "object" && !Array.isArray(value);
const fail = (message) => {
  throw new Error(message);
};
function integer(value, name, min, max) {
  if (!Number.isInteger(value) || value < min || value > max)
    fail(
      `${name} must be a whole number from ${min} to ${max}; received ${JSON.stringify(value)}.`,
    );
}
function text(value, name, maximum = 256) {
  if (
    typeof value !== "string" ||
    !value.trim() ||
    value.length > maximum ||
    /[\u0000-\u001f\u007f]/.test(value)
  )
    fail(`${name} must contain 1–${maximum} readable characters.`);
}
function storagePath(value, name) {
  text(value, name, 2048);
  if (
    !value.startsWith("/") ||
    value === "/" ||
    /^\/(?:system|vendor|product|proc|sys|dev|data\/adb|odm|apex)(?:\/|$)/.test(
      value,
    ) ||
    value.split("/").some((part) => part === "." || part === "..") ||
    value.includes("//")
  )
    fail(
      `${name} must be an absolute shared-storage path without . or .. segments; received ${JSON.stringify(value)}.`,
    );
  return value
    .replace(/\/$/, "")
    .replace(/^\/storage\/emulated\/0(?=\/|$)/, "/sdcard")
    .replace(/^\/storage\/self\/primary(?=\/|$)/, "/sdcard");
}
function overlaps(a, b) {
  return a === b || a.startsWith(`${b}/`) || b.startsWith(`${a}/`);
}
function glob(pattern, name) {
  text(pattern, name);
  if (pattern.includes("/"))
    fail(`${name} must match a filename, without folder separators.`);
  let open = false;
  for (let i = 0; i < pattern.length; i++) {
    if (pattern[i] === "\\") {
      if (++i >= pattern.length) fail(`${name} has an incomplete escape.`);
    } else if (pattern[i] === "[") {
      if (open) fail(`${name} has an invalid character range.`);
      open = true;
    } else if (pattern[i] === "]") {
      if (!open) fail(`${name} has an unmatched ].`);
      open = false;
    }
  }
  if (open) fail(`${name} has an unmatched [.`);
}
export function validateSettings(settings) {
  if (!isRecord(settings)) fail("Settings must be a JSON object.");
  if (settings.schema !== 1) fail("Settings schema must be 1.");
  for (const name of ["autoSort", "delayEnabled"])
    if (typeof settings[name] !== "boolean")
      fail(`${name} must be true or false.`);
  integer(settings.intervalSeconds, "Sort interval", 60, 604800);
  integer(settings.historyDays, "History retention", 1, 365);
  const source = storagePath(settings.source, "Source folder");
  const destination = storagePath(settings.destination, "Organization root");
  if (overlaps(source, destination))
    fail(
      "Source and organization root overlap. Neither may be inside the other.",
    );
  if (!["others", "leave"].includes(settings.unknownFiles))
    fail("Unknown-file handling must be others or leave.");
  if (
    !Array.isArray(settings.categories) ||
    !settings.categories.length ||
    settings.categories.length > 100
  )
    fail("Settings require 1–100 categories.");
  const ids = new Set();
  for (const category of settings.categories) {
    if (!isRecord(category)) fail("Each category must be an object.");
    if (
      typeof category.id !== "string" ||
      !/^[a-zA-Z0-9_-]{1,64}$/.test(category.id) ||
      ids.has(category.id)
    )
      fail(
        "Category IDs must be unique letters, numbers, hyphens or underscores.",
      );
    ids.add(category.id);
    text(category.name, "Category name", 100);
    if (typeof category.enabled !== "boolean")
      fail(`Category ${category.name}: enabled must be true or false.`);
    if (
      !Array.isArray(category.extensions) ||
      category.extensions.length > 300 ||
      category.extensions.some(
        (ext) =>
          typeof ext !== "string" ||
          !/^[a-zA-Z0-9]+(?:\.[a-zA-Z0-9]+)*$/.test(ext) ||
          ext.length > 40,
      )
    )
      fail(
        `Category ${category.name}: extensions must be a list such as pdf or tar.gz, without leading dots.`,
      );
    if (typeof category.destination !== "string" || !category.destination)
      fail(
        `Category ${category.name}: destination must be an absolute folder path.`,
      );
    if (
      category.destination &&
      overlaps(
        source,
        storagePath(
          category.destination,
          `Category ${category.name} destination`,
        ),
      )
    )
      fail(
        `Category ${category.name}: destination overlaps the source folder.`,
      );
  }
  if (!ids.has("others"))
    fail("The Others category is required in every settings document.");
  if (!Array.isArray(settings.rules) || settings.rules.length > 200)
    fail("Rules must be a list of at most 200 entries.");
  const ruleIds = new Set();
  for (const rule of settings.rules) {
    if (!isRecord(rule)) fail("Each rule must be an object.");
    if (
      typeof rule.id !== "string" ||
      !/^[a-zA-Z0-9_-]{1,64}$/.test(rule.id) ||
      ruleIds.has(rule.id)
    )
      fail("Rule IDs must be unique letters, numbers, hyphens or underscores.");
    ruleIds.add(rule.id);
    text(rule.name, "Rule name", 100);
    glob(rule.pattern, `Rule ${rule.name} pattern`);
    if (typeof rule.enabled !== "boolean")
      fail(`Rule ${rule.name}: enabled must be true or false.`);
    if (!ids.has(rule.category))
      fail(`Rule ${rule.name}: select an existing category.`);
  }
  if (!Array.isArray(settings.ignore) || settings.ignore.length > 200)
    fail("Ignore patterns must be a list of at most 200 entries.");
  settings.ignore.forEach((pattern, index) =>
    glob(pattern, `Ignore pattern ${index + 1}`),
  );
  return true;
}
export function settingsFromForm(settings, values) {
  const next = structuredClone(settings);
  for (const [key, label, min, max] of [
    ["intervalSeconds", "Sort interval", 60, 604800],
    ["historyDays", "History retention", 1, 365],
  ]) {
    const value = values[key];
    if (typeof value !== "string" || !/^\d+$/.test(value))
      fail(
        `${label} must contain whole decimal digits; received ${JSON.stringify(value)}.`,
      );
    next[key] = Number(value);
    integer(next[key], label, min, max);
  }
  validateSettings(next);
  return next;
}
export function reorder(items, index, offset) {
  const copy = [...items];
  const target = index + offset;
  if (target < 0 || target >= copy.length) return copy;
  [copy[index], copy[target]] = [copy[target], copy[index]];
  return copy;
}
function likelyOverlap(a, b) {
  a = a.toLowerCase();
  b = b.toLowerCase();
  if (a === b || a === "*" || b === "*") return true;
  // A deliberately limited check: a single wildcard's fixed prefix and suffix.
  const covers = (broad, specific) => {
    if ((broad.match(/\*/g) || []).length !== 1 || /[?\[\\]/.test(broad))
      return false;
    const [prefix, suffix] = broad.split("*");
    return specific.startsWith(prefix) && specific.endsWith(suffix);
  };
  return covers(a, b) || covers(b, a);
}
export function overlapWarnings(rules) {
  const warnings = [];
  for (let i = 0; i < rules.length; i++)
    for (let j = i + 1; j < rules.length; j++)
      if (
        rules[i].enabled &&
        rules[j].enabled &&
        likelyOverlap(rules[i].pattern, rules[j].pattern)
      )
        warnings.push(
          `“${rules[i].name}” may overlap “${rules[j].name}”. “${rules[i].name}” wins when both match.`,
        );
  return warnings;
}
export function filterPreview(items, outcome = "all", query = "") {
  const needle = query.trim().toLocaleLowerCase();
  return items.filter(
    (item) =>
      (outcome === "all" || item.outcome === outcome) &&
      (!needle ||
        [item.source, item.destination, item.reason, item.category].some(
          (value) =>
            typeof value === "string" &&
            value.toLocaleLowerCase().includes(needle),
        )),
  );
}
export const PRESETS = [
  {
    id: "ebooks",
    name: "Ebooks",
    extensions: ["epub", "mobi", "azw", "azw3", "cbz", "cbr"],
  },
  { id: "fonts", name: "Fonts", extensions: ["ttf", "otf", "woff", "woff2"] },
  {
    id: "subtitles",
    name: "Subtitles",
    extensions: ["srt", "ass", "ssa", "vtt", "sub"],
  },
  {
    id: "code",
    name: "Code",
    extensions: [
      "py",
      "sh",
      "js",
      "tsx",
      "jsx",
      "c",
      "h",
      "cpp",
      "java",
      "kt",
      "json",
      "yaml",
      "yml",
      "xml",
      "toml",
      "ini",
      "html",
      "css",
    ],
  },
  { id: "torrents", name: "Torrents", extensions: ["torrent"] },
];

export function newestRuns(runs) {
  return [...runs].sort((a, b) => b.startedAt - a.startedAt);
}
