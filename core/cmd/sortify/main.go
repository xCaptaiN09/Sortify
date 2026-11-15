package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sortify/internal/engine"
	"strconv"
	"syscall"
)

func execute() (any, error) {
	flags := flag.NewFlagSet("sortify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	state := flags.String("state", "/data/adb/sortify", "persistent state folder")
	storage := flags.String("storage", "/sdcard", "shared storage root")
	module := flags.String("module", "/data/adb/modules/sortify", "module folder")
	inherited := flags.Uint("lock-fd", 0, "inherited reserved operation lock")
	token := flags.String("token", "", "reserved operation token")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	args := flags.Args()
	if len(args) == 0 {
		return nil, fmt.Errorf("required command: init, request, request-file, run, undo or serve")
	}
	e, err := engine.New(*state, *storage, *module)
	if err != nil {
		return nil, err
	}
	e.SetToken(*token)
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(signals)
	if args[0] == "serve" {
		if len(args) != 1 {
			return nil, fmt.Errorf("serve accepts no arguments")
		}
		return nil, e.Serve(signals)
	}
	if args[0] == "run" || args[0] == "undo" {
		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case <-signals:
				_ = e.Stop()
			case <-done:
			}
		}()
	}
	var lock *engine.Lock
	if *inherited != 0 {
		if args[0] != "run" && args[0] != "undo" {
			return nil, fmt.Errorf("inherited lock is only supported for run and undo")
		}
		lock, err = e.InheritedLock(uintptr(*inherited))
		if err != nil {
			return nil, err
		}
		defer lock.Close()
	}
	switch args[0] {
	case "init":
		legacy := 0
		if len(args) > 1 {
			if len(args) != 3 || args[1] != "--legacy-interval" {
				return nil, fmt.Errorf("init accepts --legacy-interval INTEGER")
			}
			for _, r := range args[2] {
				if r < '0' || r > '9' {
					return nil, fmt.Errorf("invalid legacy interval %q", args[2])
				}
			}
			legacy, err = strconv.Atoi(args[2])
			if err != nil {
				return nil, err
			}
			if legacy < 60 || legacy > 604800 {
				return nil, fmt.Errorf("legacy interval %d must be from 60 to 604800", legacy)
			}
		}
		return e.Init(legacy)
	case "request":
		if len(args) != 2 {
			return nil, fmt.Errorf("request requires one base64 JSON argument")
		}
		return e.RequestBase64(args[1]), nil
	case "request-file":
		if len(args) != 2 {
			return nil, fmt.Errorf("request-file requires PATH or -")
		}
		var reader io.Reader = os.Stdin
		var f *os.File
		if args[1] != "-" {
			f, err = os.Open(args[1])
			if err != nil {
				return nil, err
			}
			defer f.Close()
			reader = f
		}
		raw, err := io.ReadAll(io.LimitReader(reader, 1048577))
		if err != nil {
			return nil, err
		}
		return e.RequestJSON(raw), nil
	case "run":
		if len(args) != 1 {
			return nil, fmt.Errorf("run accepts no arguments")
		}
		if lock != nil {
			return e.RunLocked()
		}
		return e.Run()
	case "undo":
		if len(args) < 2 || len(args) > 3 {
			return nil, fmt.Errorf("undo requires RUN_ID [ENTRY_ID]")
		}
		entry := ""
		if len(args) == 3 {
			entry = args[2]
		}
		if lock != nil {
			return e.UndoLocked(args[1], entry)
		}
		return e.Undo(args[1], entry)
	default:
		return nil, fmt.Errorf("unknown command %q", args[0])
	}
}
func main() {
	data, err := execute()
	response, already := data.(engine.Envelope)
	if !already {
		response = engine.Response(data, err)
	}
	if encodeErr := json.NewEncoder(os.Stdout).Encode(response); encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	if !response.OK {
		os.Exit(1)
	}
}
