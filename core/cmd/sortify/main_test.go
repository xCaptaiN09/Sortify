package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNativeProtocolAndAsynchronousReservation(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "sortify")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, output)
	}
	gate := filepath.Join(root, "search-only")
	storage := filepath.Join(gate, "storage")
	module := filepath.Join(root, "module")
	for _, p := range []string{filepath.Join(storage, "Download"), module} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(gate, 0111); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(gate, 0700) })
	flags := []string{"--state", filepath.Join(root, "state"), "--storage", storage, "--module", module}
	call := func(args ...string) map[string]json.RawMessage {
		t.Helper()
		cmd := exec.Command(binary, append(append([]string{}, flags...), args...)...)
		output, err := cmd.Output()
		var response map[string]json.RawMessage
		if decodeErr := json.Unmarshal(output, &response); decodeErr != nil {
			t.Fatalf("non-JSON output: %s (%v): %v", output, err, decodeErr)
		}
		return response
	}
	init := call("init")
	if string(init["ok"]) != "true" {
		t.Fatalf("init: %s", init["error"])
	}
	source := filepath.Join(storage, "Download", "-日本'\"\n.pdf")
	if err := os.WriteFile(source, []byte("contents"), 0600); err != nil {
		t.Fatal(err)
	}
	request := func(raw string) map[string]json.RawMessage {
		return call("request", base64.StdEncoding.EncodeToString([]byte(raw)))
	}
	bad := request(`{"command":"settings.import","settings":{"schema":1}}`)
	if string(bad["ok"]) != "false" {
		t.Fatal("invalid import did not fail")
	}
	start := request(`{"command":"start"}`)
	if string(start["ok"]) != "true" {
		t.Fatalf("start: %s", start["error"])
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		snap := request(`{"command":"snapshot"}`)
		if string(snap["ok"]) != "true" {
			t.Fatalf("snapshot: %s", snap["error"])
		}
		var state struct {
			Status struct {
				Running bool `json:"running"`
			}
			History []struct {
				State   string `json:"state"`
				Summary struct {
					Skipped int `json:"skipped"`
				}
			}
		}
		if err := json.Unmarshal(snap["data"], &state); err != nil {
			t.Fatal(err)
		}
		if !state.Status.Running {
			if len(state.History) != 1 || state.History[0].State != "complete" || state.History[0].Summary.Skipped != 1 {
				t.Fatalf("async worker failed: %s", snap["data"])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("async worker did not release reserved lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("unstable source moved: %v", err)
	}
}
