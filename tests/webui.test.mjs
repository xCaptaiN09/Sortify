import test from "node:test";
import assert from "node:assert/strict";
import { createBridge, encodeRequest } from "../webroot/bridge.mjs";
import {
  validateSettings,
  overlapWarnings,
  reorder,
  filterPreview,
  settingsFromForm,
} from "../webroot/model.mjs";

const settings = () => ({
  schema: 1,
  autoSort: false,
  intervalSeconds: 3600,
  delayEnabled: true,
  source: "/sdcard/Download",
  destination: "/sdcard/Sortify",
  historyDays: 30,
  categories: [
    {
      id: "documents",
      name: "Documents",
      enabled: true,
      extensions: ["pdf"],
      destination: "/sdcard/Sortify/Documents",
    },
    {
      id: "others",
      name: "Others",
      enabled: true,
      extensions: [],
      destination: "/sdcard/Sortify/Others",
    },
  ],
  rules: [],
  ignore: [],
  unknownFiles: "others",
});
test("interval input rejects exponent, fractions and empty values before parsing", () => {
  for (const value of ["1e3", "1.5", "", "0", "-1", "604801"])
    assert.throws(
      () =>
        settingsFromForm(settings(), {
          intervalSeconds: value,
          historyDays: "30",
        }),
      /interval/i,
    );
  assert.equal(
    settingsFromForm(settings(), { intervalSeconds: "3600", historyDays: "30" })
      .intervalSeconds,
    3600,
  );
});
test("settings reject missing categories, unsafe roots and broken rule references", () => {
  assert.throws(
    () => validateSettings({ ...settings(), source: "/data/adb" }),
    /source/i,
  );
  assert.throws(
    () =>
      validateSettings({
        ...settings(),
        destination: "/sdcard/Download/output",
      }),
    /overlap|inside/i,
  );
  assert.throws(
    () =>
      validateSettings({
        ...settings(),
        rules: [
          {
            id: "r",
            name: "Rule",
            enabled: true,
            pattern: "*",
            category: "missing",
          },
        ],
      }),
    /category/i,
  );
  assert.throws(
    () => validateSettings({ ...settings(), categories: [] }),
    /categor/i,
  );
  assert.equal(validateSettings(settings()), true);
});
test("rules expose obvious overlaps and reorder changes winner priority", () => {
  const rules = [
    {
      id: "a",
      name: "All PDFs",
      enabled: true,
      pattern: "*.pdf",
      category: "documents",
    },
    {
      id: "b",
      name: "Reports",
      enabled: true,
      pattern: "report*.pdf",
      category: "documents",
    },
  ];
  assert.match(
    overlapWarnings(rules)[0],
    /All PDFs.*Reports|Reports.*All PDFs/,
  );
  assert.deepEqual(
    reorder(rules, 1, -1).map((r) => r.id),
    ["b", "a"],
  );
  assert.deepEqual(reorder(rules, 0, -1), rules);
});
test("preview filtering retains real reasons and supports filename query", () => {
  const items = [
    { source: "/sdcard/Download/a.pdf", outcome: "move" },
    { source: "/sdcard/Download/b.pdf", outcome: "skip", reason: "Too recent" },
  ];
  assert.equal(filterPreview(items, "skip", "b.pdf")[0].reason, "Too recent");
  assert.equal(filterPreview(items, "all", "MISSING").length, 0);
});
test("bridge encodes all untrusted request values as UTF-8 base64", () => {
  const request = {
    command: "holds.release",
    path: "/sdcard/Download/é'$(touch BAD)\n.pdf",
  };
  assert.deepEqual(
    JSON.parse(Buffer.from(encodeRequest(request), "base64").toString()),
    request,
  );
});
test("callback bridge waits for errno and cleans up callback after success", async () => {
  const scope = {};
  let sent;
  scope.ksu = {
    exec(command, options, callback) {
      sent = command;
      assert.equal(options, "{}");
      queueMicrotask(() =>
        scope[callback](0, '{"ok":true,"data":{"version":"5.0"}}', ""),
      );
    },
  };
  const bridge = createBridge(scope, { timeout: 50 });
  assert.deepEqual(await bridge.request({ command: "snapshot" }), {
    version: "5.0",
  });
  assert.match(
    sent,
    /^sh \/data\/adb\/modules\/sortify\/sortify.sh request '[A-Za-z0-9+/=]+'$/,
  );
  assert.equal(Object.keys(scope).length, 1);
});
test("bridge rejects missing managers, errno, malformed responses and backend errors", async () => {
  await assert.rejects(
    createBridge({}).request({ command: "snapshot" }),
    /unsupported|KernelSU/i,
  );
  for (const [errno, stdout, message] of [
    [1, "", "exit 1"],
    [0, "not json", "JSON"],
    [0, '{"ok":false,"error":"Already running"}', "Already running"],
    [0, "{}", "envelope"],
  ]) {
    const scope = {
      ksu: {
        exec(c, o, k) {
          queueMicrotask(() => scope[k](errno, stdout, ""));
        },
      },
    };
    await assert.rejects(
      createBridge(scope).request({ command: "snapshot" }),
      new RegExp(message),
    );
  }
});
test("bridge timeout cleans callbacks and late manager callback cannot update UI", async () => {
  const scope = { ksu: { exec() {} } };
  await assert.rejects(
    createBridge(scope, { timeout: 5 }).request({ command: "snapshot" }),
    /timed out/i,
  );
  assert.equal(Object.keys(scope).length, 1);
});

test("explicit desktop demo simulates preview, start, undo and persistent protection", async () => {
  const { createDemo } = await import("../webroot/demo.mjs");
  let tick = 0;
  const demo = createDemo({ now: () => 1700000000 + tick });
  const first = await demo.request({ command: "snapshot" });
  assert.equal(first.settings.autoSort, false);
  const preview = await demo.request({ command: "preview" });
  assert(preview.items.some((item) => item.outcome === "move"));
  assert(preview.items.some((item) => item.outcome === "skip"));
  await demo.request({ command: "start" });
  assert.equal(
    (await demo.request({ command: "snapshot" })).status.running,
    true,
  );
  tick = 10;
  const finished = await demo.request({ command: "snapshot" });
  assert.equal(finished.status.running, false);
  const run = finished.history[0];
  await demo.request({ command: "undo", runId: run.id });
  tick = 20;
  const undone = await demo.request({ command: "snapshot" });
  assert(undone.holds.length > 0);
  assert(undone.history[0].undo.restored > 0);
  const path = undone.holds[0].path;
  await demo.request({ command: "history.clear" });
  assert(
    (await demo.request({ command: "snapshot" })).holds.some(
      (hold) => hold.path === path,
    ),
  );
  await demo.request({ command: "holds.release", path });
  assert(
    !(await demo.request({ command: "snapshot" })).holds.some(
      (hold) => hold.path === path,
    ),
  );
});

test("request size stays below Android single-argument limits", () => {
  assert.throws(
    () =>
      encodeRequest({
        command: "settings.import",
        settings: { value: "x".repeat(65536) },
      }),
    /64 KiB/,
  );
});
test("automatic sorting can be turned off while a demonstration run completes", async () => {
  const { createDemo } = await import("../webroot/demo.mjs");
  const demo = createDemo();
  const { settings } = await demo.request({ command: "snapshot" });
  await demo.request({
    command: "settings.save",
    settings: { ...settings, autoSort: true },
  });
  await demo.request({ command: "start" });
  await demo.request({
    command: "settings.save",
    settings: { ...settings, autoSort: false },
  });
  const snapshot = await demo.request({ command: "snapshot" });
  assert.equal(snapshot.settings.autoSort, false);
  assert.equal(snapshot.status.running, true);
});

test("overlap warnings follow case-insensitive native filename matching", () => {
  const rules = [
    {
      id: "a",
      name: "PDFs",
      enabled: true,
      pattern: "*.PDF",
      category: "documents",
    },
    {
      id: "b",
      name: "Reports",
      enabled: true,
      pattern: "report*.pdf",
      category: "documents",
    },
  ];
  assert.equal(overlapWarnings(rules).length, 1);
});

test("nonzero native exit preserves the structured backend failure explanation", async () => {
  const scope = {
    ksu: {
      exec(command, options, callback) {
        queueMicrotask(() =>
          scope[callback](
            1,
            JSON.stringify({
              ok: false,
              error: "Already running: sorting is active.",
            }),
            "",
          ),
        );
      },
    },
  };
  await assert.rejects(
    createBridge(scope).request({ command: "start" }),
    /Already running: sorting is active\./,
  );
  assert.deepEqual(Object.keys(scope), ["ksu"]);
});

test("nonzero native exit rejects a success envelope and preserves stderr for malformed output", async () => {
  for (const [stdout, stderr, expected] of [
    [JSON.stringify({ ok: true, data: { saved: true } }), "", /exit 2/],
    [
      "not JSON",
      "permission denied opening module",
      /exit 2: permission denied opening module/,
    ],
  ]) {
    const scope = {
      ksu: {
        exec(command, options, callback) {
          queueMicrotask(() => scope[callback](2, stdout, stderr));
        },
      },
    };
    await assert.rejects(
      createBridge(scope).request({ command: "snapshot" }),
      expected,
    );
  }
});

test("history sorts newest first without mutating native chronological run data", async () => {
  const { newestRuns } = await import("../webroot/model.mjs");
  const history = [
    { id: "old", startedAt: 10 },
    { id: "new", startedAt: 30 },
    { id: "middle", startedAt: 20 },
  ];
  assert.deepEqual(
    newestRuns(history).map((run) => run.id),
    ["new", "middle", "old"],
  );
  assert.deepEqual(
    history.map((run) => run.id),
    ["old", "new", "middle"],
  );
});
