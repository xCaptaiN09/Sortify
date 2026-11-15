package engine

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type Request struct {
	Command  string          `json:"command"`
	Settings json.RawMessage `json:"settings,omitempty"`
	RunID    string          `json:"runId,omitempty"`
	EntryID  string          `json:"entryId,omitempty"`
	Path     string          `json:"path,omitempty"`
}
type Envelope struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data"`
	Error string `json:"error"`
}

func Response(data any, err error) Envelope {
	if err != nil {
		return Envelope{OK: false, Error: err.Error(), Data: nil}
	}
	return Envelope{OK: true, Data: data, Error: ""}
}
func (e *Engine) RequestBase64(encoded string) Envelope {
	if len(encoded) > 1400000 {
		return Response(nil, errors.New("request exceeds 1 MiB"))
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil {
		return Response(nil, fmt.Errorf("invalid base64 request: %w", err))
	}
	return e.RequestJSON(raw)
}
func (e *Engine) RequestJSON(raw []byte) Envelope {
	var r Request
	if err := strictJSON(raw, &r); err != nil {
		return Response(nil, err)
	}
	var data any
	var err error
	switch r.Command {
	case "snapshot":
		data, err = e.Snapshot()
	case "settings.save", "settings.import":
		var s Settings
		s, err = DecodeSettings(r.Settings)
		if err == nil {
			err = e.SaveSettings(s)
		}
		if err == nil {
			data, err = e.Settings()
		}
	case "settings.export":
		data, err = e.Settings()
	case "preview":
		data, err = e.Preview()
	case "start":
		data, err = e.Launch("run", "", "")
	case "stop":
		err = e.Stop()
		data = map[string]bool{"stopping": err == nil}
	case "undo":
		if r.RunID == "" {
			err = errors.New("undo requires runId")
		} else {
			data, err = e.Launch("undo", r.RunID, r.EntryID)
		}
	case "holds.release":
		if r.Path == "" {
			err = errors.New("holds.release requires path")
		} else {
			err = e.ReleaseHold(r.Path)
			data = map[string]bool{"released": err == nil}
		}
	case "history.clear":
		err = e.ClearHistory()
		data = map[string]bool{"cleared": err == nil}
	default:
		err = fmt.Errorf("unknown command %q", r.Command)
	}
	return Response(data, err)
}
func (e *Engine) SetToken(token string) { e.token = token }
func (e *Engine) Launch(mode, runID, entryID string) (any, error) {
	if mode != "run" && mode != "undo" {
		return nil, fmt.Errorf("unsupported background operation %q", mode)
	}
	lock, err := e.acquire()
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	if _, err = e.Settings(); err != nil {
		return nil, err
	}
	db, err := e.loadDB()
	if err != nil {
		return nil, err
	}
	if mode == "undo" {
		run := findRun(db, runID)
		if run == nil {
			return nil, fmt.Errorf("run %q is unavailable or expired", runID)
		}
		if entryID != "" {
			found := false
			for _, entry := range run.Entries {
				if entry.ID == entryID {
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("entry %q is unavailable", entryID)
			}
		}
	}
	binary, err := os.Executable()
	if err != nil {
		return nil, err
	}
	token := id()
	if err = atomicJSON(filepath.Join(e.State, "active.json"), map[string]string{"runId": token}); err != nil {
		return nil, err
	}
	if err = os.Remove(filepath.Join(e.State, "stop")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	args := []string{"--state", e.State, "--storage", e.Storage, "--module", e.Module, "--lock-fd", "3", "--token", token, mode}
	if mode == "undo" {
		args = append(args, runID)
		if entryID != "" {
			args = append(args, entryID)
		}
	}
	cmd := exec.Command(binary, args...)
	cmd.ExtraFiles = []*os.File{lock.File}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	output, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer output.Close()
	cmd.Stdin = output
	cmd.Stdout = output
	cmd.Stderr = output
	previous := db.Status
	db.Status.Running = true
	db.Status.Stopping = false
	db.Status.CurrentFile = ""
	db.Status.Operation = mode
	db.Status.Error = ""
	if err = e.saveDB(db); err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		db.Status = previous
		db.Status.Error = "Background process could not start: " + err.Error()
		if saveErr := e.saveDB(db); saveErr != nil {
			return nil, fmt.Errorf("spawn failed: %v; status write failed: %w", err, saveErr)
		}
		return nil, fmt.Errorf("start background operation: %w", err)
	}
	if err = cmd.Process.Release(); err != nil {
		return nil, fmt.Errorf("operation started but process detach failed: %w", err)
	}
	if mode == "run" {
		runID = token
	}
	return map[string]any{"started": true, "runId": runID}, nil
}

// Serve polls configuration every two seconds. Runs hold their own operation
// lock, so manual and scheduled invocations share the same exclusion contract.
func (e *Engine) Serve(shutdown <-chan os.Signal) error {
	e.shutdown = shutdown
	lock, err := acquirePath(filepath.Join(e.State, "service.lock"))
	if err != nil {
		return fmt.Errorf("service: %w", err)
	}
	defer lock.Close()
	s, err := e.Settings()
	if err != nil {
		return err
	}
	next := int64(0)
	if s.AutoSort {
		next = e.now().Unix() + int64(s.IntervalSeconds)
	}
	defer atomicJSON(filepath.Join(e.State, "schedule.json"), map[string]int64{"nextRun": 0})
	writtenNext := int64(-1)
	for {
		if e.moduleStopped() {
			return nil
		}
		if _, statErr := os.Stat(e.Module); errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		select {
		case <-shutdown:
			return nil
		default:
		}
		current, err := e.Settings()
		if err != nil {
			return err
		}
		if !current.AutoSort {
			next = 0
		} else if !s.AutoSort || current.IntervalSeconds != s.IntervalSeconds || next == 0 {
			next = e.now().Unix() + int64(current.IntervalSeconds)
		}
		s = current
		if next != writtenNext {
			if err = atomicJSON(filepath.Join(e.State, "schedule.json"), map[string]int64{"nextRun": next}); err != nil {
				return err
			}
			writtenNext = next
		}
		if s.AutoSort && e.now().Unix() >= next {
			// Recheck enablement immediately before acquiring the operation lock.
			latest, lerr := e.Settings()
			if lerr != nil {
				return lerr
			}
			if latest.AutoSort {
				_, err = e.Run()
				if err != nil && !errors.Is(err, ErrBusy) {
					statusLock, lockErr := e.acquire()
					if lockErr == nil {
						db, loadErr := e.loadDB()
						if loadErr == nil {
							db.Status.Error = err.Error()
							loadErr = e.saveDB(db)
						}
						statusLock.Close()
						if loadErr != nil {
							return loadErr
						}
					} else if !errors.Is(lockErr, ErrBusy) {
						return lockErr
					}
				}
			}
			next = e.now().Unix() + int64(s.IntervalSeconds)
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-shutdown:
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}
