import { validateSettings } from "./model.mjs";

// Opt-in browser demonstration only. This adapter never reads or moves device files.
export function createDemo({ now = () => Math.floor(Date.now() / 1000) } = {}) {
  const base = "/sdcard/Download";
  const root = "/sdcard/Sortify";
  const categories = [
    [
      "documents",
      "Documents",
      [
        "pdf",
        "doc",
        "docx",
        "xls",
        "xlsx",
        "ppt",
        "pptx",
        "txt",
        "odt",
        "ods",
        "odp",
        "rtf",
        "md",
        "log",
        "tsv",
      ],
    ],
    [
      "images",
      "Images",
      [
        "jpg",
        "jpeg",
        "png",
        "gif",
        "webp",
        "heic",
        "avif",
        "tif",
        "tiff",
        "ico",
        "dng",
        "cr2",
        "nef",
        "arw",
      ],
    ],
    [
      "videos",
      "Videos",
      [
        "mp4",
        "mkv",
        "mov",
        "webm",
        "avi",
        "m4v",
        "ts",
        "mts",
        "m2ts",
        "vob",
        "wmv",
      ],
    ],
    [
      "audio",
      "Audio",
      [
        "mp3",
        "flac",
        "wav",
        "aac",
        "ogg",
        "m4a",
        "alac",
        "aiff",
        "aif",
        "ape",
        "amr",
        "mka",
        "mid",
        "midi",
      ],
    ],
    [
      "archives",
      "Archives",
      [
        "zip",
        "rar",
        "7z",
        "tar",
        "tar.gz",
        "gz",
        "bz2",
        "xz",
        "zst",
        "tgz",
        "tbz2",
        "txz",
        "lz4",
      ],
    ],
    [
      "apps",
      "Apps",
      ["apk", "apks", "apkm", "xapk", "aab", "msi", "deb", "rpm", "appimage"],
    ],
    ["others", "Others", []],
  ].map(([id, name, extensions]) => ({
    id,
    name,
    extensions,
    enabled: true,
    destination: `${root}/${name}`,
  }));
  let settings = {
    schema: 1,
    autoSort: false,
    intervalSeconds: 3600,
    delayEnabled: true,
    source: base,
    destination: root,
    historyDays: 30,
    categories,
    rules: [],
    ignore: [],
    unknownFiles: "others",
  };
  const samples = [
    { name: "Project notes.pdf" },
    { name: "Weekend in Calicut.jpg" },
    {
      name: "Project notes — copy.pdf",
      duplicateOf: `${root}/Documents/Project notes.pdf`,
    },
    { name: "Kernel sources.tar.gz" },
    { name: "New recording.m4a", recent: true },
    { name: "Firmware.zip.crdownload", partial: true },
    { name: "reading-list.epub" },
  ];
  let holds = [{ path: `${base}/Keep this receipt.pdf`, time: now() - 1200 }];
  let history = [
    {
      id: "demo-previous",
      startedAt: now() - 90000,
      finishedAt: now() - 89994,
      state: "complete",
      summary: { moved: 3, duplicates: 1, renamed: 0, skipped: 2, failed: 0 },
      entries: [
        {
          id: "previous-1",
          source: `${base}/Invoice October.pdf`,
          destination: `${root}/Documents/Invoice October.pdf`,
          category: "documents",
          outcome: "move",
          reason: "",
          duplicateOf: "",
          time: now() - 89997,
          undone: false,
        },
        {
          id: "previous-2",
          source: `${base}/DSC_0421.jpg`,
          destination: `${root}/Images/DSC_0421.jpg`,
          category: "images",
          outcome: "move",
          reason: "",
          duplicateOf: "",
          time: now() - 89996,
          undone: false,
        },
        {
          id: "previous-3",
          source: `${base}/Invoice copy.pdf`,
          destination: `${root}/Duplicates/Invoice copy.pdf`,
          category: "documents",
          outcome: "duplicate",
          reason: "Identical content",
          duplicateOf: `${root}/Documents/Invoice October.pdf`,
          time: now() - 89995,
          undone: false,
        },
      ],
    },
  ];
  let status = {
    running: false,
    stopping: false,
    serviceRunning: true,
    lastRun: history[0],
    nextRun: 0,
    currentFile: "",
    error: "",
  };
  let operation = null;
  let completed = new Set();
  let serial = 0;
  const copy = (value) => structuredClone(value);
  const glob = (pattern, name) => {
    try {
      return new RegExp(
        `^${pattern
          .replace(/[.+^$(){}|\\]/g, "\\$&")
          .replace(/\*/g, ".*")
          .replace(/\?/g, ".")}$`,
        "i",
      ).test(name);
    } catch {
      return false;
    }
  };
  function preview() {
    const items = samples
      .filter((sample) => !completed.has(sample.name))
      .map((sample) => {
        const source = `${settings.source}/${sample.name}`;
        const matched = settings.rules.filter(
          (rule) => rule.enabled && glob(rule.pattern, sample.name),
        );
        const matchingRules = matched.map((rule) => rule.id);
        const winner = matchingRules[0] || "";
        const category = matched.length
          ? settings.categories.find(
              (category) => category.id === matched[0].category,
            )
          : settings.categories
              .filter((category) => category.id !== "others")
              .flatMap((category) =>
                category.extensions.map((extension) => ({
                  category,
                  extension,
                })),
              )
              .filter((entry) =>
                sample.name
                  .toLowerCase()
                  .endsWith(`.${entry.extension.toLowerCase()}`),
              )
              .sort((a, b) => b.extension.length - a.extension.length)[0]
              ?.category;
        const selected =
          category ||
          (settings.unknownFiles === "others"
            ? settings.categories.find((category) => category.id === "others")
            : null);
        let reason = "";
        if (settings.ignore.some((pattern) => glob(pattern, sample.name)))
          reason = "Ignored by filename pattern";
        else if (sample.partial) reason = "Incomplete download";
        else if (selected && !selected.enabled) reason = "Category disabled";
        else if (!selected) reason = "Unknown extension; left in source";
        else if (sample.recent)
          reason = settings.delayEnabled
            ? "Keep recent downloads for 1 day"
            : "Too recent; stability check";
        else if (holds.some((hold) => hold.path === source))
          reason = "Restored by Undo";
        return {
          source,
          category: selected?.id || "",
          destination: reason
            ? ""
            : sample.duplicateOf
              ? `${settings.destination}/Duplicates/${sample.name}`
              : `${selected.destination || `${settings.destination}/${selected.name}`}/${sample.name}`,
          outcome: reason ? "skip" : sample.duplicateOf ? "duplicate" : "move",
          reason,
          duplicateOf: sample.duplicateOf || "",
          matchingRules,
          winner,
        };
      });
    for (const hold of holds)
      if (!items.some((item) => item.source === hold.path))
        items.push({
          source: hold.path,
          destination: "",
          category: "documents",
          outcome: "skip",
          reason: "Restored by Undo",
          duplicateOf: "",
          matchingRules: [],
          winner: "",
        });
    return { items, summary: summarize(items), warnings: [] };
  }
  function summarize(items) {
    return {
      moved: items.filter((item) =>
        ["move", "duplicate"].includes(item.outcome),
      ).length,
      duplicates: items.filter((item) => item.outcome === "duplicate").length,
      renamed: 0,
      skipped: items.filter((item) => item.outcome === "skip").length,
      failed: items.filter((item) => item.outcome === "error").length,
    };
  }
  function finish() {
    if (!operation || now() < operation.finishAt) return;
    if (operation.type === "sort") {
      const items = operation.stopped
        ? operation.items.slice(0, 2)
        : operation.items;
      const entries = items.map((item, index) => ({
        ...item,
        id: `entry-${index}`,
        time: now(),
        undone: false,
      }));
      const run = {
        id: `demo-${++serial}`,
        startedAt: operation.startedAt,
        finishedAt: now(),
        state: operation.stopped ? "stopped" : "complete",
        summary: summarize(entries),
        entries,
      };
      history.unshift(run);
      status.lastRun = run;
      entries
        .filter((entry) => ["move", "duplicate"].includes(entry.outcome))
        .forEach((entry) => completed.add(entry.source.split("/").pop()));
    } else {
      const run = history.find((run) => run.id === operation.runId);
      const entries = run.entries.filter(
        (entry) =>
          (!operation.entryId || entry.id === operation.entryId) &&
          !entry.undone &&
          ["move", "duplicate"].includes(entry.outcome),
      );
      entries.forEach((entry) => {
        entry.undone = true;
        holds.push({ path: entry.source, time: now() });
        completed.delete(entry.source.split("/").pop());
      });
      run.undo = {
        restored: entries.length,
        conflicted: 0,
        failed: 0,
        entries: copy(entries),
      };
      status.lastRun = run;
    }
    status.running = false;
    status.stopping = false;
    status.currentFile = "";
    operation = null;
  }
  return {
    supported: true,
    async request(request) {
      finish();
      switch (request.command) {
        case "snapshot":
          return copy({
            version: "5.0",
            settings,
            status: {
              ...status,
              nextRun: settings.autoSort ? now() + settings.intervalSeconds : 0,
            },
            history,
            holds,
            warnings: [],
          });
        case "preview":
          if (status.running)
            throw new Error(
              "Already running. Wait for this operation to finish.",
            );
          return copy(preview());
        case "settings.save":
        case "settings.import":
          validateSettings(request.settings);
          settings = copy(request.settings);
          return copy(settings);
        case "settings.export":
          return copy(settings);
        case "start":
          if (status.running) throw new Error("Already running.");
          operation = {
            type: "sort",
            items: preview().items,
            startedAt: now(),
            finishAt: now() + 3,
          };
          status.running = true;
          status.currentFile = operation.items[0]?.source || "";
          return { started: true };
        case "stop":
          if (!operation) throw new Error("No operation is running.");
          operation.stopped = true;
          operation.finishAt = now() + 1;
          status.stopping = true;
          return { stopping: true };
        case "undo": {
          if (status.running) throw new Error("Already running.");
          const run = history.find((run) => run.id === request.runId);
          if (!run) throw new Error("Run not found.");
          operation = {
            type: "undo",
            runId: request.runId,
            entryId: request.entryId,
            startedAt: now(),
            finishAt: now() + 3,
          };
          status.running = true;
          status.currentFile = "Restoring files";
          return { started: true };
        }
        case "holds.release":
          holds = holds.filter((hold) => hold.path !== request.path);
          return { released: true };
        case "history.clear":
          if (status.running) throw new Error("Already running.");
          history = [];
          return { cleared: true };
        default:
          throw new Error(
            `Unsupported demonstration command: ${request.command}`,
          );
      }
    },
  };
}
