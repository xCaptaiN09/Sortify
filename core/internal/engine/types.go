package engine

import (
	"os"
	"time"
)

const Version = "5.0"

type Category struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Enabled     bool     `json:"enabled"`
	Extensions  []string `json:"extensions"`
	Destination string   `json:"destination"`
}
type Rule struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Pattern  string `json:"pattern"`
	Category string `json:"category"`
}
type Settings struct {
	Schema          int        `json:"schema"`
	AutoSort        bool       `json:"autoSort"`
	IntervalSeconds int        `json:"intervalSeconds"`
	DelayEnabled    bool       `json:"delayEnabled"`
	Source          string     `json:"source"`
	Destination     string     `json:"destination"`
	HistoryDays     int        `json:"historyDays"`
	Categories      []Category `json:"categories"`
	Rules           []Rule     `json:"rules"`
	Ignore          []string   `json:"ignore"`
	UnknownFiles    string     `json:"unknownFiles"`
}
type Identity struct {
	Device   uint64 `json:"device"`
	Inode    uint64 `json:"inode"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
	Changed  int64  `json:"changed"`
	Hash     string `json:"hash"`
}
type Observation struct {
	Identity Identity `json:"identity"`
	Since    int64    `json:"since"`
}
type Hold struct {
	Reason   string   `json:"reason,omitempty"`
	Path     string   `json:"path"`
	Identity Identity `json:"identity"`
	Time     int64    `json:"time"`
}
type Summary struct {
	Moved      int `json:"moved"`
	Duplicates int `json:"duplicates"`
	Renamed    int `json:"renamed"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}
type Entry struct {
	ID            string   `json:"id"`
	Source        string   `json:"source"`
	Destination   string   `json:"destination"`
	Category      string   `json:"category"`
	Outcome       string   `json:"outcome"`
	Reason        string   `json:"reason"`
	DuplicateOf   string   `json:"duplicateOf"`
	MatchingRules []string `json:"matchingRules"`
	Winner        string   `json:"winner"`
	Time          int64    `json:"time"`
	Undone        bool     `json:"undone"`
	Identity      Identity `json:"identity"`
	Renamed       bool     `json:"renamed"`
}
type Run struct {
	ID         string      `json:"id"`
	StartedAt  int64       `json:"startedAt"`
	FinishedAt int64       `json:"finishedAt"`
	State      string      `json:"state"`
	Summary    Summary     `json:"summary"`
	Entries    []Entry     `json:"entries"`
	Error      string      `json:"error,omitempty"`
	Undo       *UndoResult `json:"undo,omitempty"`
}
type Preview struct {
	Items    []Entry  `json:"items"`
	Summary  Summary  `json:"summary"`
	Warnings []string `json:"warnings"`
}
type Status struct {
	Running        bool   `json:"running"`
	Stopping       bool   `json:"stopping"`
	ServiceRunning bool   `json:"serviceRunning"`
	LastRun        *Run   `json:"lastRun"`
	NextRun        int64  `json:"nextRun"`
	CurrentFile    string `json:"currentFile"`
	Error          string `json:"error"`
	Operation      string `json:"operation"`
}
type Snapshot struct {
	Version  string   `json:"version"`
	Settings Settings `json:"settings"`
	Status   Status   `json:"status"`
	History  []Run    `json:"history"`
	Holds    []Hold   `json:"holds"`
	Warnings []string `json:"warnings"`
}
type UndoResult struct {
	Restored   int     `json:"restored"`
	Conflicted int     `json:"conflicted"`
	Failed     int     `json:"failed"`
	Entries    []Entry `json:"entries"`
}
type Intent struct {
	Source              string   `json:"source"`
	Destination         string   `json:"destination"`
	Staging             string   `json:"staging"`
	Identity            Identity `json:"identity"`
	DestinationIdentity Identity `json:"destinationIdentity"`
	RunID               string   `json:"runId"`
	EntryID             string   `json:"entryId"`
	Phase               string   `json:"phase"`
	Undo                bool     `json:"undo"`
	Entry               Entry    `json:"entry"`
}
type database struct {
	History      []Run                  `json:"history"`
	Holds        []Hold                 `json:"holds"`
	Observations map[string]Observation `json:"observations"`
	Journal      *Intent                `json:"journal"`
	Warnings     []string               `json:"warnings"`
	Status       Status                 `json:"status"`
}
type Engine struct {
	State, Storage, Module string
	now                    func() time.Time
	token                  string
	shutdown               <-chan os.Signal
	interrupted            bool
}
