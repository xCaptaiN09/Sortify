package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (e *Engine) Run() (Run, error) {
	lock, err := e.acquire()
	if err != nil {
		return Run{}, err
	}
	defer lock.Close()
	return e.RunLocked()
}
func (e *Engine) prepare() (*database, Settings, error) {
	s, err := e.Settings()
	if err != nil {
		return nil, s, err
	}
	db, err := e.loadDB()
	if err != nil {
		return nil, s, err
	}
	if err = e.recover(db); err != nil {
		return nil, s, err
	}
	for i := range db.History {
		if db.History[i].FinishedAt == 0 {
			db.History[i].FinishedAt = e.now().Unix()
			db.History[i].State = "interrupted"
		}
	}
	return db, s, nil
}
func (e *Engine) prune(db *database, days int) {
	cutoff := e.now().Unix() - int64(days)*86400
	keep := []Run{}
	for _, r := range db.History {
		if r.FinishedAt == 0 || r.FinishedAt >= cutoff {
			keep = append(keep, r)
		}
	}
	if len(keep) > 200 {
		keep = keep[len(keep)-200:]
	}
	db.History = keep
	if len(db.Warnings) > 50 {
		db.Warnings = db.Warnings[len(db.Warnings)-50:]
	}
}
func (e *Engine) currentStopped(runID string) bool {
	data, err := os.ReadFile(filepath.Join(e.State, "stop"))
	return err == nil && string(data) == runID
}
func (e *Engine) moduleStopped() bool {
	if e.interrupted {
		return true
	}
	if _, err := os.Stat(e.Module); errors.Is(err, os.ErrNotExist) {
		return true
	}
	if e.shutdown != nil {
		select {
		case <-e.shutdown:
			e.interrupted = true
			return true
		default:
		}
	}
	for _, flag := range []string{"disable", "remove"} {
		if _, err := os.Stat(filepath.Join(e.Module, flag)); err == nil {
			return true
		}
	}
	return false
}
func (e *Engine) RunLocked() (result Run, err error) {
	db, s, err := e.prepare()
	if err != nil {
		return result, err
	}
	e.prune(db, s.HistoryDays)
	token := e.token
	if token == "" {
		token = id()
	}
	result = Run{ID: token, StartedAt: e.now().Unix(), State: "running", Entries: []Entry{}}
	db.History = append(db.History, result)
	index := len(db.History) - 1
	db.Status = Status{Running: true, Operation: "sort", CurrentFile: ""}
	db.Status.Error = ""
	if e.token == "" {
		if err = os.Remove(filepath.Join(e.State, "stop")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return result, err
		}
	}
	if err = atomicJSON(filepath.Join(e.State, "active.json"), map[string]string{"runId": result.ID}); err != nil {
		return result, err
	}
	db.Status.Error = ""
	if err = e.saveDB(db); err != nil {
		return result, err
	}
	defer func() {
		result.FinishedAt = e.now().Unix()
		if err != nil {
			result.Error = err.Error()
			if result.Summary.Moved == 0 {
				result.State = "failed"
			} else {
				result.State = "completed_with_errors"
			}
		} else if result.State == "running" {
			if result.Summary.Failed > 0 {
				result.State = "completed_with_errors"
			} else {
				result.State = "complete"
			}
		}
		db.History[index] = result
		db.Status.Running = false
		db.Status.Stopping = false
		db.Status.CurrentFile = ""
		db.Status.LastRun = &result
		db.Status.Error = result.Error
		if saveErr := e.saveDB(db); saveErr != nil {
			err = fmt.Errorf("final history write failed: %w", saveErr)
		}
	}()
	source, err := e.scanSource(s)
	if err != nil {
		return result, err
	}
	candidates, err := e.managed(s)
	if err != nil {
		return result, err
	}
	reserved := map[string]bool{}
	live := map[string]bool{}
	for _, de := range source {
		if de.IsDir() {
			continue
		}
		if e.currentStopped(result.ID) || e.moduleStopped() {
			result.State = "stopped"
			break
		}
		db.Status.CurrentFile = filepath.Join(s.Source, de.Name())
		db.History[index] = result
		if err = e.saveDB(db); err != nil {
			return result, err
		}
		item := e.eligibility(s, db, de)
		live[item.Source] = true
		item, err = e.route(item, s, candidates, reserved)
		if err != nil {
			item.Outcome = "error"
			item.Reason = err.Error()
			err = nil
		}
		if item.Outcome == "move" || item.Outcome == "duplicate" {
			intent := &Intent{Source: item.Source, Destination: item.Destination, Identity: item.Identity, RunID: result.ID, EntryID: item.ID, Phase: "copying", Entry: item}
			var dest Identity
			dest, err = e.transfer(db, intent)
			if err != nil {
				item.Outcome = "error"
				item.Reason = err.Error()
				result.Entries = append(result.Entries, item)
				addSummary(&result.Summary, item)
				return result, err
			}
			item.Identity = dest
			result.Entries = append(result.Entries, item)
			addSummary(&result.Summary, item)
			delete(db.Observations, item.Source)
			candidates = append(candidates, item.Destination)
			reserved[strings.ToLower(item.Destination)] = true
			db.Journal = nil
		} else {
			result.Entries = append(result.Entries, item)
			addSummary(&result.Summary, item)
		}
		db.History[index] = result
		if err = e.saveDB(db); err != nil {
			return result, err
		}
	}
	// Keep only direct source observations. A stopped scan retains observations for
	// files not reached yet so stopping cannot reset their one-day delay.
	if result.State != "stopped" {
		for path := range db.Observations {
			if !live[path] {
				delete(db.Observations, path)
			}
		}
	}
	return result, nil
}
func (e *Engine) Undo(runID, entryID string) (UndoResult, error) {
	lock, err := e.acquire()
	if err != nil {
		return UndoResult{}, err
	}
	defer lock.Close()
	return e.UndoLocked(runID, entryID)
}
func (e *Engine) UndoLocked(runID, entryID string) (result UndoResult, err error) {
	result.Entries = []Entry{}
	db, _, err := e.prepare()
	if err != nil {
		return result, err
	}
	run := findRun(db, runID)
	if run == nil {
		return result, fmt.Errorf("run %q is unavailable or expired", runID)
	}
	found := entryID == ""
	for _, entry := range run.Entries {
		if entry.ID == entryID {
			found = true
		}
	}
	if !found {
		return result, fmt.Errorf("entry %q is unavailable in run %q", entryID, runID)
	}
	token := e.token
	if token == "" {
		token = id()
	}
	if err = atomicJSON(filepath.Join(e.State, "active.json"), map[string]string{"runId": token}); err != nil {
		return result, err
	}
	db.Status.Running = true
	db.Status.Operation = "undo"
	db.Status.Error = ""
	if err = e.saveDB(db); err != nil {
		return result, err
	}
	defer func() {
		run = findRun(db, runID)
		if run != nil {
			run.Undo = &result
		}
		db.Status.Running = false
		db.Status.Stopping = false
		db.Status.CurrentFile = ""
		if err != nil {
			db.Status.Error = err.Error()
		}
		if saveErr := e.saveDB(db); saveErr != nil {
			err = saveErr
		}
	}()
	for i := len(run.Entries) - 1; i >= 0; i-- {
		if e.currentStopped(token) || e.moduleStopped() {
			break
		}
		entry := run.Entries[i]
		if entryID != "" && entry.ID != entryID {
			continue
		}
		if entry.Undone || (entry.Outcome != "move" && entry.Outcome != "duplicate") {
			continue
		}
		db.Status.CurrentFile = entry.Destination
		if err = e.saveDB(db); err != nil {
			return result, err
		}
		current, ferr := fingerprint(entry.Destination)
		if ferr != nil || !sameIdentity(current, entry.Identity) || !sameContent(current, entry.Identity) {
			entry.Reason = "Undo conflict: moved file changed, disappeared or was replaced"
			result.Conflicted++
			result.Entries = append(result.Entries, entry)
			continue
		}
		// Historical roots are still required to remain inside the validated shared
		// storage. They need not equal today's configured source or destinations.
		if _, ferr = e.validatePath(filepath.Dir(entry.Source)); ferr != nil {
			entry.Reason = ferr.Error()
			result.Failed++
			result.Entries = append(result.Entries, entry)
			continue
		}
		if _, ferr = e.validatePath(filepath.Dir(entry.Destination)); ferr != nil {
			entry.Reason = ferr.Error()
			result.Failed++
			result.Entries = append(result.Entries, entry)
			continue
		}
		_, ferr = os.Lstat(entry.Source)
		if ferr == nil || !errors.Is(ferr, os.ErrNotExist) {
			entry.Reason = "Undo conflict: original path is occupied or inaccessible"
			result.Conflicted++
			result.Entries = append(result.Entries, entry)
			continue
		}
		// Case-equivalent names are conflicts even on case-sensitive test storage.
		name, renamed, nerr := freeName(filepath.Dir(entry.Source), filepath.Base(entry.Source), map[string]bool{})
		if nerr != nil || renamed || name != entry.Source {
			entry.Reason = "Undo conflict: original filename is occupied"
			result.Conflicted++
			result.Entries = append(result.Entries, entry)
			continue
		}
		intent := &Intent{Source: entry.Destination, Destination: entry.Source, Identity: current, RunID: runID, EntryID: entry.ID, Phase: "copying", Undo: true, Entry: entry}
		restored, ferr := e.transfer(db, intent)
		if ferr != nil {
			entry.Reason = ferr.Error()
			result.Failed++
			result.Entries = append(result.Entries, entry)
			return result, ferr
		}
		run = findRun(db, runID)
		run.Entries[i].Undone = true
		run.Entries[i].Reason = "Restored by Undo"
		entry.Undone = true
		entry.Reason = "Restored by Undo"
		db.Holds = append(db.Holds, Hold{Path: entry.Source, Identity: restored, Time: e.now().Unix()})
		db.Observations[entry.Source] = Observation{Identity: restored, Since: e.now().Unix()}
		db.Journal = nil
		result.Restored++
		result.Entries = append(result.Entries, entry)
		if err = e.saveDB(db); err != nil {
			return result, err
		}
	}
	return result, nil
}
func (e *Engine) Stop() error {
	lock, err := e.acquire()
	if err == nil {
		lock.Close()
		return errors.New("No operation is running")
	}
	if !errors.Is(err, ErrBusy) {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(e.State, "active.json"))
	if err != nil {
		return err
	}
	var active struct {
		RunID string `json:"runId"`
	}
	if err = strictJSON(raw, &active); err != nil {
		return err
	}
	if active.RunID == "" {
		return errors.New("operation is starting; retry Stop")
	}
	tmp := filepath.Join(e.State, ".stop-"+id())
	if err = os.WriteFile(tmp, []byte(active.RunID), 0600); err != nil {
		return err
	}
	defer os.Remove(tmp)
	return os.Rename(tmp, filepath.Join(e.State, "stop"))
}
func (e *Engine) ReleaseHold(path string) error {
	p, err := e.validatePath(filepath.Dir(path))
	if err != nil {
		return err
	}
	path = filepath.Join(p, filepath.Base(path))
	lock, err := e.acquire()
	if err != nil {
		return err
	}
	defer lock.Close()
	db, err := e.loadDB()
	if err != nil {
		return err
	}
	holds := []Hold{}
	found := false
	for _, h := range db.Holds {
		if h.Path == path {
			found = true
		} else {
			holds = append(holds, h)
		}
	}
	if !found {
		return fmt.Errorf("no Undo hold exists for %q", path)
	}
	db.Holds = holds
	return e.saveDB(db)
}
func (e *Engine) ClearHistory() error {
	lock, err := e.acquire()
	if err != nil {
		return err
	}
	defer lock.Close()
	db, err := e.loadDB()
	if err != nil {
		return err
	}
	if db.Journal != nil {
		return errors.New("recover the interrupted operation before clearing history")
	}
	db.History = []Run{}
	db.Status.LastRun = nil
	return e.saveDB(db)
}
func (e *Engine) Snapshot() (Snapshot, error) {
	s, err := e.Settings()
	if err != nil {
		return Snapshot{}, err
	}
	db, err := e.loadDB()
	if err != nil {
		return Snapshot{}, err
	}
	status := db.Status
	lock, lerr := e.acquire()
	if lerr == nil {
		status.Running = false
		status.Stopping = false
		status.CurrentFile = ""
		lock.Close()
	} else if errors.Is(lerr, ErrBusy) {
		status.Running = true
	} else {
		return Snapshot{}, lerr
	}
	if status.Running {
		var active struct {
			RunID string `json:"runId"`
		}
		if raw, err := os.ReadFile(filepath.Join(e.State, "active.json")); err == nil && strictJSON(raw, &active) == nil {
			status.Stopping = e.currentStopped(active.RunID)
		}
	}
	service, serr := acquirePath(filepath.Join(e.State, "service.lock"))
	if serr == nil {
		service.Close()
		status.ServiceRunning = false
	} else if errors.Is(serr, ErrBusy) {
		status.ServiceRunning = true
	} else {
		return Snapshot{}, serr
	}
	var sched struct {
		NextRun int64 `json:"nextRun"`
	}
	if raw, err := os.ReadFile(filepath.Join(e.State, "schedule.json")); err == nil {
		if err = strictJSON(raw, &sched); err != nil {
			return Snapshot{}, err
		}
		if status.ServiceRunning && s.AutoSort {
			status.NextRun = sched.NextRun
		} else {
			status.NextRun = 0
		}
	}
	status.LastRun = nil
	for i := len(db.History) - 1; i >= 0; i-- {
		if db.History[i].FinishedAt > 0 {
			copy := db.History[i]
			status.LastRun = &copy
			break
		}
	}

	warn := append(warnings(s), db.Warnings...)
	if db.Journal != nil {
		warn = append(warn, "Interrupted file operation awaits recovery on the next Sort Now or Undo.")
	}
	return Snapshot{Version: Version, Settings: s, Status: status, History: db.History, Holds: db.Holds, Warnings: warn}, nil
}
