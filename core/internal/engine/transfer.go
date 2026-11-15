package engine

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func capturedIdentity(a, b Identity) bool {
	return a.Device == b.Device && a.Inode == b.Inode && a.Size == b.Size && a.Modified == b.Modified && sameContent(a, b)
}
func (e *Engine) transfer(db *database, intent *Intent) (Identity, error) {
	if db.Journal != nil {
		return Identity{}, errors.New("unrecovered file operation")
	}
	db.Journal = intent
	if err := e.saveDB(db); err != nil {
		return Identity{}, err
	}
	if err := secureMkdir(filepath.Dir(intent.Destination)); err != nil {
		return Identity{}, err
	}
	source, err := secureOpen(intent.Source, os.O_RDONLY, 0)
	if err != nil {
		return Identity{}, err
	}
	defer source.Close()
	st, err := source.Stat()
	if err != nil {
		return Identity{}, err
	}
	si, err := identityInfo(st)
	if err != nil || !sameIdentity(si, intent.Identity) {
		return Identity{}, errors.New("source changed before copy")
	}
	dest, err := secureOpen(intent.Destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, uint32(st.Mode().Perm()))
	if err != nil {
		return Identity{}, fmt.Errorf("destination %q: %w", intent.Destination, err)
	}
	_, copyErr := io.Copy(dest, source)
	if copyErr == nil {
		tv := syscall.NsecToTimeval(intent.Identity.Modified)
		copyErr = syscall.Futimes(int(dest.Fd()), []syscall.Timeval{tv, tv})
	}
	if copyErr == nil {
		copyErr = dest.Sync()
	}
	closeErr := dest.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return Identity{}, fmt.Errorf("copy failed; source retained and partial destination preserved at %q: %w", intent.Destination, copyErr)
	}
	di, err := fingerprint(intent.Destination)
	if err != nil {
		return Identity{}, err
	}
	if !sameContent(di, intent.Identity) {
		return Identity{}, errors.New("copied content verification failed; source retained")
	}
	same, err := equalBytes(intent.Source, intent.Destination)
	if err != nil || !same {
		if err == nil {
			err = errors.New("byte verification failed")
		}
		return Identity{}, err
	}
	current, err := fingerprint(intent.Source)
	if err != nil {
		return Identity{}, err
	}
	if !sameIdentity(current, intent.Identity) || !sameContent(current, intent.Identity) {
		return Identity{}, errors.New("source changed during copy; source and destination retained")
	}
	if err = syncDir(filepath.Dir(intent.Destination)); err != nil {
		return Identity{}, err
	}
	intent.DestinationIdentity = di
	intent.Phase = "verified"
	if err = e.saveDB(db); err != nil {
		return Identity{}, err
	}
	// Capture the source atomically in a dedicated staging directory before deleting it. If
	// another app replaced the path, the captured identity check preserves it.
	stageDir, stage, err := privateStage(filepath.Dir(intent.Source))
	if err != nil {
		return Identity{}, err
	}
	defer stage.Close()
	intent.Staging = filepath.Join(stageDir, "source")
	if err = e.saveDB(db); err != nil {
		return Identity{}, err
	}
	if err = capturePath(intent.Source, stage); err != nil {
		return Identity{}, err
	}
	if err = syncDir(filepath.Dir(intent.Source)); err != nil {
		return Identity{}, err
	}
	captured, err := fingerprint(intent.Staging)
	if err != nil {
		return Identity{}, err
	}
	if !capturedIdentity(captured, intent.Identity) {
		return Identity{}, fmt.Errorf("source was replaced during move; replacement preserved at %q; verified copy at %q", intent.Staging, intent.Destination)
	}
	// The verified destination must still be the same object and contents before
	// destroying the captured source. This also catches concurrent external edits.
	final, err := fingerprint(intent.Destination)
	if err != nil {
		return Identity{}, err
	}
	if !sameIdentity(final, di) || !sameContent(final, di) {
		return Identity{}, fmt.Errorf("destination changed; original preserved at %q", intent.Staging)
	}
	if knownWriter(captured) {
		return Identity{}, fmt.Errorf("writer remains open; original preserved at %q", intent.Staging)
	}
	intent.Phase = "captured"
	if err = e.saveDB(db); err != nil {
		return Identity{}, err
	}
	destNow, destErr := statIdentity(intent.Destination)
	if destErr != nil || !sameIdentity(destNow, final) {
		return Identity{}, fmt.Errorf("destination changed before source removal; source preserved at %q", intent.Staging)
	}
	finalSource, checkErr := statIdentity(intent.Staging)
	if checkErr != nil || !sameIdentity(finalSource, captured) {
		return Identity{}, fmt.Errorf("source changed before removal; original preserved at %q", intent.Staging)
	}
	if err = guardedRemove(stage, "source", captured); err != nil {
		return Identity{}, err
	}
	if err = syncDir(stageDir); err != nil {
		return Identity{}, err
	}
	if err = os.Remove(stageDir); err != nil {
		return Identity{}, err
	}
	if err = syncDir(filepath.Dir(intent.Source)); err != nil {
		return Identity{}, err
	}
	intent.Phase = "done"
	if err = e.saveDB(db); err != nil {
		return Identity{}, err
	}
	return final, nil
}
func findRun(db *database, id string) *Run {
	for i := range db.History {
		if db.History[i].ID == id {
			return &db.History[i]
		}
	}
	return nil
}
func (e *Engine) finishIntent(db *database, intent *Intent, dest Identity, recovered bool) {
	run := findRun(db, intent.RunID)
	if run == nil {
		db.History = append(db.History, Run{ID: intent.RunID, StartedAt: e.now().Unix(), State: "completed_with_errors", Entries: []Entry{}})
		run = &db.History[len(db.History)-1]
	}
	if intent.Undo {
		for i := range run.Entries {
			if run.Entries[i].ID == intent.EntryID {
				run.Entries[i].Undone = true
				run.Entries[i].Reason = "Restored by Undo"
			}
		}
		db.Holds = append(db.Holds, Hold{Path: intent.Destination, Identity: dest, Time: e.now().Unix()})
		entry := intent.Entry
		entry.Undone = true
		entry.Reason = "Restored by Undo"
		upsertUndo(run, entry)
	} else {
		entry := intent.Entry
		entry.Identity = dest
		entry.Time = e.now().Unix()
		if recovered {
			entry.Reason = "Recovered completed move after interruption"
		}
		found := false
		for i, item := range run.Entries {
			if item.ID == intent.EntryID {
				run.Entries[i] = entry
				found = true
				break
			}
		}
		if !found {
			run.Entries = append(run.Entries, entry)
		}
		run.Summary = Summary{}
		for _, item := range run.Entries {
			addSummary(&run.Summary, item)
		}

	}
}
func (e *Engine) recover(db *database) error {
	j := db.Journal
	if j == nil {
		return nil
	}
	// Validate every journal path again; corrupt state must never target system files.
	for _, p := range []string{j.Source, j.Destination} {
		if _, err := e.validatePath(filepath.Dir(p)); err != nil {
			return fmt.Errorf("unsafe journal: %w", err)
		}
	}
	if j.Staging != "" {
		if filepath.Dir(filepath.Dir(j.Staging)) != filepath.Dir(j.Source) || filepath.Base(j.Staging) != "source" {
			return errors.New("unsafe staging path in journal")
		}
		if _, err := e.validatePath(filepath.Dir(j.Staging)); err != nil {
			return err
		}
	}
	dest, derr := fingerprint(j.Destination)
	_, serr := os.Lstat(j.Source)
	completed := errors.Is(serr, os.ErrNotExist) && derr == nil && sameContent(dest, j.Identity) && sameIdentity(dest, j.DestinationIdentity)
	if j.Staging != "" {
		stage, stageErr := secureOpen(filepath.Dir(j.Staging), os.O_RDONLY|directoryFlag, 0)
		if stageErr == nil {
			defer stage.Close()
		} else if !errors.Is(stageErr, os.ErrNotExist) {
			return stageErr
		}
		captured, cerr := fingerprint(j.Staging)
		if cerr == nil {
			completed = false
			if derr == nil && sameIdentity(dest, j.DestinationIdentity) && sameContent(dest, j.Identity) && capturedIdentity(captured, j.Identity) {
				destAgain, verifyErr := fingerprint(j.Destination)
				if verifyErr != nil || !sameIdentity(destAgain, dest) || !sameContent(destAgain, dest) {
					return fmt.Errorf("recovery destination changed; source preserved at %q", j.Staging)
				}
				if knownWriter(captured) {
					return fmt.Errorf("writer remains open; interrupted source preserved at %q", j.Staging)
				}
				destNow, destErr := statIdentity(j.Destination)
				if destErr != nil || !sameIdentity(destNow, destAgain) {
					return fmt.Errorf("destination changed before recovery removal; source preserved at %q", j.Staging)
				}
				finalSource, checkErr := statIdentity(j.Staging)
				if checkErr != nil || !sameIdentity(finalSource, captured) {
					return fmt.Errorf("captured source changed; preserved at %q", j.Staging)
				}
				if err := guardedRemove(stage, "source", captured); err != nil {
					return err
				}
				if err := syncDir(filepath.Dir(j.Staging)); err != nil {
					return err
				}
				if err := os.Remove(filepath.Dir(j.Staging)); err != nil {
					return err
				}
				if err := syncDir(filepath.Dir(j.Source)); err != nil {
					return err
				}
				completed = true
			} else {
				db.Warnings = append(db.Warnings, fmt.Sprintf("Interrupted move preserved source at %q and destination at %q; inspect both before retrying.", j.Staging, j.Destination))
			}
		} else if !errors.Is(cerr, os.ErrNotExist) {
			return cerr
		}
	}
	if completed {
		e.finishIntent(db, j, dest, true)
	} else {
		message := fmt.Sprintf("Interrupted operation: source %q and destination %q preserved wherever present. Unverified copies are never removed automatically.", j.Source, j.Destination)
		db.Warnings = append(db.Warnings, message)
		if source, ferr := fingerprint(j.Source); ferr == nil {
			db.Holds = append(db.Holds, Hold{Path: j.Source, Identity: source, Time: e.now().Unix(), Reason: "Interrupted operation: inspect preserved copies, then allow sorting again"})
		}
		if run := findRun(db, j.RunID); run != nil {
			entry := j.Entry
			entry.Outcome = "error"
			entry.Reason = message
			if j.Undo {
				upsertUndo(run, entry)
			} else {
				found := false
				for i, previous := range run.Entries {
					if previous.ID == entry.ID {
						run.Entries[i] = entry
						found = true
						break
					}
				}
				if !found {
					run.Entries = append(run.Entries, entry)
				}
				run.Summary = Summary{}
				for _, item := range run.Entries {
					addSummary(&run.Summary, item)
				}
			}
		}
	}
	db.Journal = nil
	return e.saveDB(db)
}

func upsertUndo(run *Run, entry Entry) {
	if run.Undo == nil {
		run.Undo = &UndoResult{Entries: []Entry{}}
	}
	found := false
	for i, previous := range run.Undo.Entries {
		if previous.ID == entry.ID {
			run.Undo.Entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		run.Undo.Entries = append(run.Undo.Entries, entry)
	}
	run.Undo.Restored = 0
	run.Undo.Conflicted = 0
	run.Undo.Failed = 0
	for _, item := range run.Undo.Entries {
		if item.Undone {
			run.Undo.Restored++
		} else if strings.HasPrefix(item.Reason, "Undo conflict:") {
			run.Undo.Conflicted++
		} else {
			run.Undo.Failed++
		}
	}
}
