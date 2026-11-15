package engine

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestStorageTraversalThroughSearchOnlyParents(t *testing.T) {
	root := t.TempDir()
	gate := filepath.Join(root, "search-only")
	storage := filepath.Join(gate, "storage")
	source := filepath.Join(storage, "Download")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source, "document.pdf")
	if err := os.WriteFile(path, []byte("readable download"), 0644); err != nil {
		t.Fatal(err)
	}
	e, err := New(filepath.Join(root, "state"), storage, filepath.Join(root, "module"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(gate, 0111); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(gate, 0755) })
	if _, err := os.ReadDir(gate); err == nil {
		t.Skip("test requires an account without permission to list the search-only parent")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "readable download" {
		t.Fatalf("fixture must allow direct child access: data=%q error=%v", data, err)
	}
	t.Run("scan source", func(t *testing.T) {
		entries, err := e.scanSource(e.Defaults())
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "document.pdf" {
			t.Fatalf("unexpected source entries: %v", entries)
		}
	})
	t.Run("read file", func(t *testing.T) {
		if _, err := fingerprint(path); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("create destination", func(t *testing.T) {
		destination := filepath.Join(storage, "Sortify", "Documents")
		if err := secureMkdir(destination); err != nil {
			t.Fatal(err)
		}
		if info, err := os.Stat(destination); err != nil || !info.IsDir() {
			t.Fatalf("destination not created: %v", err)
		}
	})
	t.Run("final access remains required", func(t *testing.T) {
		if err := os.Chmod(path, 0000); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(path, 0644) }()
		if _, err := fingerprint(path); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("expected final-file permission failure, got %v", err)
		}
		if err := os.Chmod(source, 0000); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(source, 0755) }()
		if _, err := e.scanSource(e.Defaults()); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("expected final-directory permission failure, got %v", err)
		}
	})
	t.Run("ancestor symlinks remain rejected", func(t *testing.T) {
		link := filepath.Join(storage, "redirect")
		if err := os.Symlink(source, link); err != nil {
			t.Fatal(err)
		}
		if f, err := secureOpen(filepath.Join(link, "document.pdf"), os.O_RDONLY, 0); err == nil {
			f.Close()
			t.Fatal("opened a file through a replaced symlink ancestor")
		}
		if err := secureMkdir(filepath.Join(link, "new")); err == nil {
			t.Fatal("created a directory through a replaced symlink ancestor")
		}
	})
}

func fixture(t *testing.T) *Engine {
	t.Helper()
	root := t.TempDir()
	storage := filepath.Join(root, "storage")
	if err := os.MkdirAll(filepath.Join(storage, "Download"), 0700); err != nil {
		t.Fatal(err)
	}
	e, err := New(filepath.Join(root, "state"), storage, filepath.Join(root, "module"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(e.Module, 0700); err != nil {
		t.Fatal(err)
	}
	e.now = func() time.Time { return time.Unix(1800000000, 0) }
	if _, err = e.Init(0); err != nil {
		t.Fatal(err)
	}
	s, err := e.Settings()
	if err != nil {
		t.Fatal(err)
	}
	s.DelayEnabled = false
	if err = e.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	return e
}
func put(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Unix(1700000000, 0)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
}
func settle(t *testing.T, e *Engine) {
	t.Helper()
	if _, err := e.Preview(); err != nil {
		t.Fatal(err)
	}
	now := e.now()
	e.now = func() time.Time { return now.Add(61 * time.Second) }
}
func TestDefaultsAndAtomicValidation(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	if s.AutoSort || s.IntervalSeconds != 3600 || s.HistoryDays != 30 {
		t.Fatalf("bad defaults: %+v", s)
	}
	before, _ := os.ReadFile(e.configPath())
	s.IntervalSeconds = 0
	if e.SaveSettings(s) == nil {
		t.Fatal("accepted invalid interval")
	}
	after, _ := os.ReadFile(e.configPath())
	if string(before) != string(after) {
		t.Fatal("invalid save changed config")
	}
	s, _ = e.Settings()
	s.Destination = s.Source
	if e.SaveSettings(s) == nil {
		t.Fatal("accepted source/destination overlap")
	}
}
func TestStableObservationAndDelay(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	p := filepath.Join(s.Source, "fresh.pdf")
	put(t, p, "a")
	r, err := e.Run()
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary.Moved != 0 || r.Summary.Skipped != 1 {
		t.Fatalf("first observation moved: %+v", r)
	}
	now := e.now()
	e.now = func() time.Time { return now.Add(61 * time.Second) }
	r, err = e.Run()
	if err != nil || r.Summary.Moved != 1 {
		t.Fatalf("stable file not moved: %+v %v", r, err)
	}
}
func TestDuplicatesCollisionAndSpecialNames(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	put(t, filepath.Join(s.Destination, "Documents", "report.pdf"), "old")
	put(t, filepath.Join(s.Destination, "Documents", "existing.pdf"), "same")
	put(t, filepath.Join(s.Destination, "Duplicates", "copy.pdf"), "occupied")
	put(t, filepath.Join(s.Source, "report.pdf"), "new")
	put(t, filepath.Join(s.Source, "copy.pdf"), "same")
	put(t, filepath.Join(s.Source, "-quotes'\"\n日本.pdf"), "special")
	settle(t, e)
	r, err := e.Run()
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary.Moved != 3 || r.Summary.Duplicates != 1 || r.Summary.Renamed != 2 {
		t.Fatalf("wrong result %+v entries %+v", r.Summary, r.Entries)
	}
	for p, want := range map[string]string{filepath.Join(s.Destination, "Documents", "report.pdf"): "old", filepath.Join(s.Destination, "Documents", "report (1).pdf"): "new", filepath.Join(s.Destination, "Duplicates", "copy.pdf"): "occupied", filepath.Join(s.Destination, "Duplicates", "copy (1).pdf"): "same"} {
		b, err := os.ReadFile(p)
		if err != nil || string(b) != want {
			t.Fatalf("%s: %q %v", p, b, err)
		}
	}
}
func TestDisabledIgnoreRulesAndLongestSuffix(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	s.Ignore = []string{"secret*"}
	s.Rules = []Rule{{ID: "all", Name: "All PDFs", Enabled: true, Pattern: "*.pdf", Category: "images"}, {ID: "specific", Name: "Specific", Enabled: true, Pattern: "a*", Category: "documents"}}
	for i := range s.Categories {
		if s.Categories[i].ID == "images" {
			s.Categories[i].Enabled = false
		}
	}
	if err := e.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.pdf", "secret.txt", "part.PART", ".hidden.pdf"} {
		put(t, filepath.Join(s.Source, n), n)
	}
	settle(t, e)
	p, err := e.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 4 {
		t.Fatalf("items %+v", p.Items)
	}
	for _, i := range p.Items {
		if i.Outcome != "skip" {
			t.Fatalf("unexpected eligibility %+v", i)
		}
		if strings.HasSuffix(i.Source, "a.pdf") && (i.Winner != "all" || len(i.MatchingRules) != 2) {
			t.Fatalf("rule priority %+v", i)
		}
	}
}
func TestUndoHoldsAndReplacement(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	src := filepath.Join(s.Source, "a.pdf")
	put(t, src, "one")
	settle(t, e)
	r, err := e.Run()
	if err != nil {
		t.Fatal(err)
	}
	u, err := e.Undo(r.ID, "")
	if err != nil || u.Restored != 1 {
		t.Fatalf("undo %+v %v", u, err)
	}
	if err := e.ClearHistory(); err != nil {
		t.Fatal(err)
	}
	p, err := e.Preview()
	if err != nil || p.Items[0].Reason != "Restored by Undo" {
		t.Fatalf("hold lost %+v %v", p, err)
	}
	if err = e.ReleaseHold(src); err != nil {
		t.Fatal(err)
	}
	settle(t, e)
	r, err = e.Run()
	if err != nil || r.Summary.Moved != 1 {
		t.Fatalf("release %+v %v", r, err)
	}
	put(t, src, "occupied")
	u, err = e.Undo(r.ID, "")
	if err != nil || u.Conflicted != 1 {
		t.Fatalf("overwrite undo %+v %v", u, err)
	}
}
func TestSymlinkRejectedAndSingleLock(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	outside := filepath.Join(t.TempDir(), "private")
	put(t, outside, "secret")
	if err := os.Symlink(outside, filepath.Join(s.Source, "link.pdf")); err != nil {
		t.Fatal(err)
	}
	settle(t, e)
	r, err := e.Run()
	if err != nil || r.Summary.Moved != 0 {
		t.Fatalf("symlink moved %+v %v", r, err)
	}
	l, err := e.acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if _, err = e.Run(); err == nil {
		t.Fatal("second operation acquired lock")
	}
}
func TestStrictJSONAndRecovery(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	raw, _ := json.Marshal(s)
	raw = []byte(strings.Replace(string(raw), `"intervalSeconds":3600`, `"intervalSeconds":3.6e3`, 1))
	if _, err := DecodeSettings(raw); err == nil {
		t.Fatal("accepted exponent interval")
	}
	src := filepath.Join(s.Source, "interrupted.pdf")
	dst := filepath.Join(s.Destination, "Documents", "interrupted.pdf")
	put(t, src, "intact")
	settle(t, e)
	identity, err := fingerprint(src)
	if err != nil {
		t.Fatal(err)
	}
	db, err := e.loadDB()
	if err != nil {
		t.Fatal(err)
	}
	db.Journal = &Intent{Source: src, Destination: dst, Identity: identity, RunID: "interrupted", EntryID: "entry", Phase: "copying"}
	if err = e.saveDB(db); err != nil {
		t.Fatal(err)
	}
	put(t, dst, "partial")
	if _, err = e.Run(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(src)
	if string(b) != "intact" {
		t.Fatal("recovery removed uncertain source")
	}
	b, _ = os.ReadFile(dst)
	if string(b) != "partial" {
		t.Fatal("recovery removed uncertain destination")
	}
}

func TestOneDayDelayBoundaryAndChangedObservation(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	s.DelayEnabled = true
	if err := e.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(s.Source, "old.pdf")
	put(t, p, "old")
	settle(t, e)
	preview, err := e.Preview()
	if err != nil || !strings.Contains(preview.Items[0].Reason, "1 day") {
		t.Fatalf("one-day delay bypassed: %+v %v", preview, err)
	}
	start := e.now()
	e.now = func() time.Time { return start.Add(86338 * time.Second) }
	preview, err = e.Preview()
	if err != nil || preview.Items[0].Outcome != "skip" {
		t.Fatalf("delay expired early %+v %v", preview, err)
	}
	e.now = func() time.Time { return start.Add(86339 * time.Second) }
	preview, err = e.Preview()
	if err != nil || preview.Items[0].Outcome != "move" {
		t.Fatalf("delay boundary %+v %v", preview, err)
	}
	put(t, p, "changed")
	preview, err = e.Preview()
	if err != nil || !strings.Contains(preview.Items[0].Reason, "Still changing") {
		t.Fatalf("changed identity did not reset observation %+v %v", preview, err)
	}
}
func TestDuplicateOnlyInDuplicatesAndCaseCollision(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	put(t, filepath.Join(s.Destination, "Duplicates", "already.pdf"), "same")
	put(t, filepath.Join(s.Destination, "Documents", "REPORT.PDF"), "different")
	put(t, filepath.Join(s.Source, "report.pdf"), "unique")
	put(t, filepath.Join(s.Source, "else.pdf"), "same")
	settle(t, e)
	r, err := e.Run()
	if err != nil || r.Summary.Moved != 2 || r.Summary.Duplicates != 1 || r.Summary.Renamed != 1 {
		t.Fatalf("case/duplicate %+v %v", r, err)
	}
	if _, err = os.Stat(filepath.Join(s.Destination, "Documents", "report (1).pdf")); err != nil {
		t.Fatal(err)
	}
}
func TestSameNameSameContentPreservesBothAndUndoChangedConflict(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	put(t, filepath.Join(s.Destination, "Documents", "a.pdf"), "same")
	put(t, filepath.Join(s.Source, "a.pdf"), "same")
	settle(t, e)
	r, err := e.Run()
	if err != nil || r.Summary.Duplicates != 1 {
		t.Fatalf("duplicate %+v %v", r, err)
	}
	put(t, r.Entries[0].Destination, "changed")
	u, err := e.Undo(r.ID, r.Entries[0].ID)
	if err != nil || u.Conflicted != 1 || u.Restored != 0 {
		t.Fatalf("changed undo %+v %v", u, err)
	}
}
func TestUndoReplacementDoesNotInheritHold(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	p := filepath.Join(s.Source, "a.pdf")
	put(t, p, "one")
	settle(t, e)
	r, err := e.Run()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Undo(r.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(p); err != nil {
		t.Fatal(err)
	}
	put(t, p, "two")
	preview, err := e.Preview()
	if err != nil || preview.Items[0].Reason == "Restored by Undo" {
		t.Fatalf("reused filename held %+v %v", preview, err)
	}
	settle(t, e)
	r, err = e.Run()
	if err != nil || r.Summary.Moved != 1 {
		t.Fatalf("replacement blocked %+v %v", r, err)
	}
}
func TestSettingsSaveWhileOperationLockedAndStrictCompleteness(t *testing.T) {
	e := fixture(t)
	lock, err := e.acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	s, _ := e.Settings()
	s.AutoSort = true
	if err = e.SaveSettings(s); err != nil {
		t.Fatalf("cannot configure active run: %v", err)
	}
	raw, _ := json.Marshal(s)
	for _, bad := range []string{strings.Replace(string(raw), `"autoSort":true,`, "", 1), strings.Replace(string(raw), `"delayEnabled":false`, `"delayEnabled":null`, 1), strings.Replace(string(raw), `"enabled":true`, `"enabled":null`, 1)} {
		if _, err = DecodeSettings([]byte(bad)); err == nil {
			t.Fatalf("accepted incomplete/null settings %s", bad)
		}
	}
}
func TestRecoveredCompletedMoveReplacesUncertainHistory(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	src := filepath.Join(s.Source, "a.pdf")
	dst := filepath.Join(s.Destination, "Documents", "a.pdf")
	put(t, dst, "one")
	di, err := fingerprint(dst)
	if err != nil {
		t.Fatal(err)
	}
	entry := Entry{ID: "e", Source: src, Destination: dst, Outcome: "move", Identity: di}
	db, _ := e.loadDB()
	bad := entry
	bad.Outcome = "error"
	db.History = []Run{{ID: "r", State: "failed", Summary: Summary{Failed: 1}, Entries: []Entry{bad}}}
	db.Journal = &Intent{Source: src, Destination: dst, Identity: di, DestinationIdentity: di, RunID: "r", EntryID: "e", Phase: "done", Entry: entry}
	if err = e.saveDB(db); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Run(); err != nil {
		t.Fatal(err)
	}
	snap, err := e.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	r := snap.History[0]
	if r.Summary.Moved != 1 || r.Summary.Failed != 0 || r.Entries[0].Outcome != "move" {
		t.Fatalf("recovered move not accounted %+v", r)
	}
}
func TestStopCurrentOperationThenNextRun(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	for i := 0; i < 30; i++ {
		put(t, filepath.Join(s.Source, strings.Repeat("a", i+1)+".pdf"), strings.Repeat("x", 65536)+string(rune(i)))
	}
	settle(t, e)
	done := make(chan Run, 1)
	failure := make(chan error, 1)
	go func() { r, err := e.Run(); done <- r; failure <- err }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		snap, err := e.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if snap.Status.Running && snap.Status.CurrentFile != "" {
			if err = e.Stop(); err != nil {
				t.Fatal(err)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("run did not become active")
		}
		time.Sleep(time.Millisecond)
	}
	r := <-done
	if err := <-failure; err != nil {
		t.Fatal(err)
	}
	if r.State != "stopped" || r.Summary.Moved < 1 || r.Summary.Moved >= 30 {
		t.Fatalf("stop did not finish current file only %+v", r)
	}
	r, err := e.Run()
	if err != nil || r.State == "stopped" || r.Summary.Moved == 0 {
		t.Fatalf("stale stop request affected next run %+v %v", r, err)
	}
}

func TestRejectProtectedStorageAndSymlinkDestination(t *testing.T) {
	for _, root := range []string{"/", "/data", "/data/adb", "/system", "/proc", "/etc"} {
		if _, err := New(filepath.Join(t.TempDir(), "state"), root, filepath.Join(t.TempDir(), "module")); err == nil {
			t.Errorf("accepted protected storage %q", root)
		}
	}
	e := fixture(t)
	s, _ := e.Settings()
	outside := t.TempDir()
	link := filepath.Join(e.Storage, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	s.Categories[0].Destination = filepath.Join(link, "Documents")
	if err := e.SaveSettings(s); err == nil {
		t.Fatal("accepted destination symlink escaping shared storage")
	}
}
func TestUndoRunPartialConflictAndIndependentRetention(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	for _, n := range []string{"a.pdf", "b.pdf", "c.pdf"} {
		put(t, filepath.Join(s.Source, n), n)
	}
	settle(t, e)
	r, err := e.Run()
	if err != nil {
		t.Fatal(err)
	}
	put(t, r.Entries[0].Source, "newer")
	u, err := e.Undo(r.ID, "")
	if err != nil || u.Restored != 2 || u.Conflicted != 1 {
		t.Fatalf("partial undo %+v %v", u, err)
	}
	e2, err := New(e.State, e.Storage, e.Module)
	if err != nil {
		t.Fatal(err)
	}
	e2.now = e.now
	if err = e2.ClearHistory(); err != nil {
		t.Fatal(err)
	}
	snap, err := e2.Snapshot()
	if err != nil || len(snap.History) != 0 || len(snap.Holds) != 2 {
		t.Fatalf("persistent holds %+v %v", snap, err)
	}
}
func TestPreviewPreservesSourceAndHistoryAndLongestSuffix(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	s.Categories = append(s.Categories, Category{ID: "compound", Name: "Compound", Enabled: true, Extensions: []string{"tar.gz"}, Destination: filepath.Join(s.Destination, "Compound")})
	for i := range s.Categories {
		if s.Categories[i].ID == "archives" {
			s.Categories[i].Extensions = []string{"gz"}
		}
	}
	if err := e.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(s.Source, "backup.TAR.GZ")
	put(t, src, "archive")
	settle(t, e)
	p, err := e.Preview()
	if err != nil || p.Items[0].Category != "compound" || p.Items[0].Winner != "extension:tar.gz" {
		t.Fatalf("suffix %+v %v", p, err)
	}
	if _, err = os.Stat(src); err != nil {
		t.Fatal(err)
	}
	snap, err := e.Snapshot()
	if err != nil || len(snap.History) != 0 {
		t.Fatalf("preview changed history %+v %v", snap, err)
	}
}
func TestImportInvalidCompleteDocumentLeavesConfigUnchanged(t *testing.T) {
	e := fixture(t)
	before, _ := os.ReadFile(e.configPath())
	r := e.RequestJSON([]byte(`{"command":"settings.import","settings":{"schema":1}}`))
	if r.OK {
		t.Fatal("import accepted incomplete settings")
	}
	after, _ := os.ReadFile(e.configPath())
	if string(before) != string(after) {
		t.Fatal("invalid import changed settings")
	}
}

func TestOpenWriterDefersEvenOldStableFile(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	src := filepath.Join(s.Source, "paused.pdf")
	put(t, src, "paused")
	settle(t, e)
	writer, err := os.OpenFile(src, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	r, err := e.Run()
	if err != nil || r.Summary.Moved != 0 || r.Summary.Skipped != 1 || !strings.Contains(strings.ToLower(r.Entries[0].Reason), "writer") {
		t.Fatalf("open writer moved %+v %v", r, err)
	}
}

func TestRecoveryDoesNotDoubleCountFailedEntry(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	src := filepath.Join(s.Source, "a.pdf")
	dst := filepath.Join(s.Destination, "Documents", "a.pdf")
	put(t, src, "good")
	put(t, dst, "partial")
	si, err := fingerprint(src)
	if err != nil {
		t.Fatal(err)
	}
	entry := Entry{ID: "e", Source: src, Destination: dst, Outcome: "move", Identity: si}
	bad := entry
	bad.Outcome = "error"
	db, _ := e.loadDB()
	db.History = []Run{{ID: "r", State: "failed", FinishedAt: e.now().Unix(), Summary: Summary{Failed: 1}, Entries: []Entry{bad}}}
	db.Journal = &Intent{Source: src, Destination: dst, Identity: si, RunID: "r", EntryID: "e", Phase: "copying", Entry: entry}
	if err = e.saveDB(db); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Run(); err != nil {
		t.Fatal(err)
	}
	snap, err := e.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	r := snap.History[0]
	if len(r.Entries) != 1 || r.Summary.Failed != 1 {
		t.Fatalf("recovery double-counted entry %+v", r)
	}
}

func TestSchedulerReloadAndSignalStopsActiveRunThenExits(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	s.AutoSort = true
	s.IntervalSeconds = 60
	if err := e.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.pdf", "b.pdf", "c.pdf"} {
		put(t, filepath.Join(s.Source, n), n)
	}
	settle(t, e)
	var clock atomic.Int64
	clock.Store(e.now().Unix())
	e.now = func() time.Time { return time.Unix(clock.Load(), 0) }
	signals := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() { done <- e.Serve(signals) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		snap, err := e.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if snap.Status.ServiceRunning && snap.Status.NextRun > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("service did not start")
		}
		time.Sleep(time.Millisecond)
	}
	clock.Add(61)
	deadline = time.Now().Add(5 * time.Second)
	for {
		snap, err := e.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if snap.Status.Running && snap.Status.CurrentFile != "" {
			signals <- syscall.SIGTERM
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scheduled run did not start")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("service consumed shutdown but kept running")
	}
	snap, err := e.Snapshot()
	if err != nil || snap.Status.ServiceRunning || snap.Status.Running {
		t.Fatalf("service did not stop %+v %v", snap.Status, err)
	}
	if len(snap.History) != 1 || snap.History[0].State != "stopped" || snap.History[0].Summary.Moved < 1 || snap.History[0].Summary.Moved >= 3 {
		t.Fatalf("service stopped at wrong boundary %+v", snap.History)
	}
}

func TestOverlapWarningsUseCaseInsensitiveRuleMatching(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	s.Rules = []Rule{{ID: "first", Name: "First", Enabled: true, Pattern: "*.PDF", Category: "documents"}, {ID: "second", Name: "Second", Enabled: true, Pattern: "*.pdf", Category: "images"}}
	if len(warnings(s)) != 1 {
		t.Fatalf("case-equivalent overlap not warned: %+v", warnings(s))
	}
}

func TestBoundStageValidationNeverDeletesSubstitutedFile(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	put(t, filepath.Join(a, "source"), "original")
	expected, err := statIdentity(filepath.Join(a, "source"))
	if err != nil {
		t.Fatal(err)
	}
	stage, err := secureOpen(a, os.O_RDONLY|directoryFlag, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	if err = os.Rename(a, b); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(a, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(filepath.Join(b, "source"), filepath.Join(a, "source")); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(b, "source"), "replacement")
	if err = guardedRemove(stage, "source", expected); err == nil {
		t.Fatal("removed unrelated bound-directory child")
	}
	body, err := os.ReadFile(filepath.Join(b, "source"))
	if err != nil || string(body) != "replacement" {
		t.Fatalf("replacement destroyed: %q %v", body, err)
	}
}

func TestCaptureRefusesToReplaceStagingFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "incoming")
	put(t, source, "incoming")
	stagePath, stage, err := privateStage(root)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	put(t, filepath.Join(stagePath, "source"), "existing")
	if err = capturePath(source, stage); err == nil {
		t.Fatal("capture overwrote an existing staging file")
	}
	for p, want := range map[string]string{source: "incoming", filepath.Join(stagePath, "source"): "existing"} {
		body, err := os.ReadFile(p)
		if err != nil || string(body) != want {
			t.Fatalf("capture changed %q: %q %v", p, body, err)
		}
	}
}

func TestRecoveryDoesNotDoubleCountFailedUndo(t *testing.T) {
	e := fixture(t)
	s, _ := e.Settings()
	original := filepath.Join(s.Source, "a.pdf")
	organized := filepath.Join(s.Destination, "Documents", "a.pdf")
	put(t, organized, "good")
	put(t, original, "partial")
	identity, err := fingerprint(organized)
	if err != nil {
		t.Fatal(err)
	}
	entry := Entry{ID: "e", Source: original, Destination: organized, Outcome: "move", Identity: identity}
	bad := entry
	bad.Reason = "failed restore"
	db, _ := e.loadDB()
	db.History = []Run{{ID: "r", Entries: []Entry{entry}, Undo: &UndoResult{Failed: 1, Entries: []Entry{bad}}}}
	db.Journal = &Intent{Source: organized, Destination: original, Identity: identity, RunID: "r", EntryID: "e", Phase: "copying", Undo: true, Entry: entry}
	if err = e.recover(db); err != nil {
		t.Fatal(err)
	}
	if db.History[0].Undo.Failed != 1 || len(db.History[0].Undo.Entries) != 1 || db.History[0].Entries[0].Outcome != "move" {
		t.Fatalf("undo recovery double-counted %+v", db.History[0])
	}
}
