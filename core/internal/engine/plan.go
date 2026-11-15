package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var temporarySuffixes = []string{".part", ".partial", ".crdownload", ".download", ".tmp", ".temp", ".opdownload", ".filepart", ".aria2", ".!qb", ".!ut"}

func (e *Engine) classify(s Settings, name string) (Category, []string, string) {
	matches := []string{}
	winner := ""
	target := ""
	for _, r := range s.Rules {
		if !r.Enabled {
			continue
		}
		match, _ := filepath.Match(strings.ToLower(r.Pattern), strings.ToLower(name))
		if match {
			matches = append(matches, r.ID)
			if winner == "" {
				winner = r.ID
				target = r.Category
			}
		}
	}
	if target == "" {
		longest := 0
		for _, c := range s.Categories {
			for _, ext := range c.Extensions {
				if strings.HasSuffix(strings.ToLower(name), "."+ext) && len(ext) > longest {
					target = c.ID
					longest = len(ext)
					winner = "extension:" + ext
				}
			}
		}
	}
	if target == "" {
		if s.UnknownFiles == "leave" {
			return Category{}, matches, "unknown"
		}
		target = "others"
		winner = "fallback"
	}
	for _, c := range s.Categories {
		if c.ID == target {
			return c, matches, winner
		}
	}
	return Category{}, matches, winner
}
func (e *Engine) managed(s Settings) ([]string, error) {
	roots := []string{filepath.Join(s.Destination, "Duplicates")}
	for _, c := range s.Categories {
		roots = append(roots, c.Destination)
	}
	sort.Strings(roots)
	out := []string{}
	seen := map[string]bool{}
	for _, root := range roots {
		if _, err := secureOpenDir(root); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				if p != root && strings.HasPrefix(d.Name(), ".sortify-") {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan managed folder %q: %w", root, err)
		}
	}
	return out, nil
}
func secureOpenDir(p string) (bool, error) {
	f, err := secureOpen(p, os.O_RDONLY|directoryFlag, 0)
	if err != nil {
		return false, err
	}
	return true, f.Close()
}
func (e *Engine) scanSource(s Settings) ([]os.DirEntry, error) {
	f, err := secureOpen(s.Source, os.O_RDONLY|directoryFlag, 0)
	if err != nil {
		return nil, fmt.Errorf("source unavailable: %w", err)
	}
	defer f.Close()
	items, err := f.ReadDir(-1)
	sort.Slice(items, func(i, j int) bool { return items[i].Name() < items[j].Name() })
	return items, err
}
func (e *Engine) eligibility(s Settings, db *database, de os.DirEntry) Entry {
	path := filepath.Join(s.Source, de.Name())
	item := Entry{ID: id(), Source: path, Outcome: "skip", MatchingRules: []string{}, Time: e.now().Unix()}
	if de.IsDir() {
		item.Reason = "Subfolder left untouched"
		return item
	}
	if !de.Type().IsRegular() {
		item.Reason = "Symlink or special file left untouched"
		return item
	}
	name := de.Name()
	lower := strings.ToLower(name)
	if strings.HasPrefix(name, ".") {
		item.Reason = "Hidden file"
		return item
	}
	for _, suffix := range temporarySuffixes {
		if strings.HasSuffix(lower, suffix) {
			item.Reason = "Incomplete download"
			return item
		}
	}
	for _, pattern := range s.Ignore {
		match, _ := filepath.Match(strings.ToLower(pattern), lower)
		if match {
			item.Reason = "Ignored: " + pattern
			return item
		}
	}
	category, matches, winner := e.classify(s, name)
	item.Category = category.ID
	item.MatchingRules = matches
	item.Winner = winner
	if category.ID == "" {
		item.Reason = "Unknown file left in source"
		return item
	}
	if !category.Enabled {
		item.Reason = "Category disabled"
		return item
	}
	identity, err := statIdentity(path)
	if err != nil {
		item.Outcome = "error"
		item.Reason = err.Error()
		return item
	}
	item.Identity = identity
	if knownWriter(identity) {
		item.Reason = "Still changing: an open writer descriptor or shared writable mapping was detected"
		return item
	}
	for _, hold := range db.Holds {
		if hold.Path != path {
			continue
		}
		if sameIdentity(identity, hold.Identity) {
			current, err := fingerprint(path)
			if err != nil {
				item.Outcome = "error"
				item.Reason = err.Error()
				return item
			}
			if sameContent(current, hold.Identity) {
				item.Reason = hold.Reason
				if item.Reason == "" {
					item.Reason = "Restored by Undo"
				}
				return item
			}
		}
	}
	observation, seen := db.Observations[path]
	if !seen || !sameIdentity(observation.Identity, identity) {
		observation = Observation{Identity: identity, Since: e.now().Unix()}
		db.Observations[path] = observation
		item.Reason = "Still changing: waiting for a separate stability observation"
		return item
	}
	age := e.now().Sub(time.Unix(0, identity.Modified))
	if age < 60*time.Second || e.now().Unix()-observation.Since < 60 {
		item.Reason = "Too recent: minimum 60 second stability check"
		return item
	}
	if s.DelayEnabled && (age < 24*time.Hour || e.now().Unix()-observation.Since < 86400) {
		item.Reason = "Keep recent downloads for 1 day (estimated from observation and modification time)"
		return item
	}
	item.Outcome = "move"
	item.Reason = ""
	item.Destination = filepath.Join(category.Destination, name)
	return item
}
func (e *Engine) route(item Entry, s Settings, candidates []string, reserved map[string]bool) (Entry, error) {
	if item.Outcome != "move" {
		return item, nil
	}
	current, err := fingerprint(item.Source)
	if err != nil {
		return item, err
	}
	if !sameIdentity(current, item.Identity) {
		return item, errors.New("source changed before comparison")
	}
	item.Identity = current
	for _, p := range candidates {
		candidate, err := statIdentity(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return item, fmt.Errorf("compare %q: %w", p, err)
		}
		if candidate.Size != current.Size {
			continue
		}
		candidate, err = fingerprint(p)
		if err != nil {
			return item, fmt.Errorf("compare %q: %w", p, err)
		}
		if candidate.Hash != current.Hash {
			continue
		}
		same, err := equalBytes(item.Source, p)
		if err != nil {
			return item, err
		}
		after, err := statIdentity(p)
		if err != nil {
			return item, err
		}
		if !sameIdentity(candidate, after) {
			return item, fmt.Errorf("duplicate candidate changed: %q", p)
		}
		sourceAfter, err := statIdentity(item.Source)
		if err != nil {
			return item, err
		}
		if !sameIdentity(current, sourceAfter) {
			return item, errors.New("source changed during comparison")
		}
		if same {
			item.Outcome = "duplicate"
			item.DuplicateOf = p
			item.Destination = filepath.Join(s.Destination, "Duplicates", filepath.Base(item.Source))
			break
		}
	}
	dir := filepath.Dir(item.Destination)
	if _, err = e.validatePath(dir); err != nil {
		return item, err
	}
	item.Destination, item.Renamed, err = freeName(dir, filepath.Base(item.Source), reserved)
	return item, err
}
func addSummary(s *Summary, item Entry) {
	switch item.Outcome {
	case "move", "duplicate":
		s.Moved++
		if item.Outcome == "duplicate" {
			s.Duplicates++
		}
		if item.Renamed {
			s.Renamed++
		}
	case "skip":
		s.Skipped++
	case "error":
		s.Failed++
	}
}
func (e *Engine) Preview() (Preview, error) {
	lock, err := e.acquire()
	if err != nil {
		return Preview{}, err
	}
	defer lock.Close()
	s, err := e.Settings()
	if err != nil {
		return Preview{}, err
	}
	db, err := e.loadDB()
	if err != nil {
		return Preview{}, err
	}
	if db.Journal != nil {
		return Preview{}, errors.New("an interrupted operation needs recovery; run Sort Now before preview")
	}
	source, err := e.scanSource(s)
	if err != nil {
		return Preview{}, err
	}
	candidates, err := e.managed(s)
	if err != nil {
		return Preview{}, err
	}
	result := Preview{Items: []Entry{}, Warnings: append(warnings(s), db.Warnings...)}
	reserved := map[string]bool{}
	live := map[string]bool{}
	for _, de := range source {
		if de.IsDir() {
			continue
		}
		item := e.eligibility(s, db, de)
		live[item.Source] = true
		item, err = e.route(item, s, candidates, reserved)
		if err != nil {
			item.Outcome = "error"
			item.Reason = err.Error()
		}
		if item.Outcome == "move" || item.Outcome == "duplicate" {
			reserved[strings.ToLower(item.Destination)] = true
			// Compare later incoming files against earlier eligible source contents without moving either.
			candidates = append(candidates, item.Source)
		}
		result.Items = append(result.Items, item)
		addSummary(&result.Summary, item)
	}
	for path := range db.Observations {
		if !live[path] {
			delete(db.Observations, path)
		}
	}
	if err = e.saveDB(db); err != nil {
		return Preview{}, err
	}
	return result, nil
}
