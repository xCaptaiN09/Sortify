package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func New(state, storage, module string) (*Engine, error) {
	for label, p := range map[string]string{"state": state, "storage": storage, "module": module} {
		if !filepath.IsAbs(p) {
			return nil, fmt.Errorf("%s path must be absolute: %q", label, p)
		}
	}
	root, err := canonical(storage)
	if err != nil {
		return nil, fmt.Errorf("storage %q: %w", storage, err)
	}
	st, err := canonical(state)
	if err != nil {
		return nil, err
	}
	mod, err := canonical(module)
	if err != nil {
		return nil, err
	}
	if root == "/" || within(root, st) || within(st, root) || within(root, mod) || within(mod, root) {
		return nil, errors.New("storage, state and module must be separate folders")
	}
	for _, protected := range []string{"/system", "/vendor", "/product", "/proc", "/sys", "/dev", "/data", "/odm", "/apex", "/etc", "/usr", "/boot", "/root", "/bin", "/sbin", "/lib"} {
		if within(protected, root) {
			return nil, fmt.Errorf("protected storage path %q", root)
		}
	}
	return &Engine{State: st, Storage: root, Module: mod, now: time.Now}, nil
}
func (e *Engine) configPath() string { return filepath.Join(e.State, "config.json") }
func (e *Engine) Defaults() Settings {
	s := Settings{Schema: 1, IntervalSeconds: 3600, DelayEnabled: true, Source: filepath.Join(e.Storage, "Download"), Destination: filepath.Join(e.Storage, "Sortify"), HistoryDays: 30, UnknownFiles: "others", Rules: []Rule{}, Ignore: []string{}}
	presets := []struct {
		id, name, ext string
		enabled       bool
	}{
		{"documents", "Documents", "pdf doc docx xls xlsx ppt pptx txt odt ods odp rtf md log csv tsv", true},
		{"images", "Images", "jpg jpeg png gif webp bmp heic heif avif tif tiff ico dng cr2 nef arw svg", true},
		{"videos", "Videos", "mp4 mkv avi mov webm 3gp flv m4v ts mts m2ts vob wmv mpeg mpg", true},
		{"audio", "Audio", "mp3 m4a aac flac wav ogg opus wma alac aiff aif ape amr mka mid midi", true},
		{"archives", "Archives", "zip rar 7z tar gz tar.gz bz2 tar.bz2 xz tar.xz zst tar.zst tgz tbz2 txz lz4", true},
		{"apps", "Apps", "apk apks apkm xapk aab exe msi deb rpm appimage", true},
		{"ebooks", "Ebooks", "epub mobi azw azw3 cbz cbr", false},
		{"fonts", "Fonts", "ttf otf woff woff2", false},
		{"subtitles", "Subtitles", "srt ass ssa vtt sub", false},
		{"code", "Code", "py sh js tsx jsx c h cpp java kt json yaml yml xml toml ini html css", false},
		{"torrents", "Torrents", "torrent", false}, {"others", "Others", "", true},
	}
	for _, p := range presets {
		ext := strings.Fields(p.ext)
		if ext == nil {
			ext = []string{}
		}
		s.Categories = append(s.Categories, Category{ID: p.id, Name: p.name, Enabled: p.enabled, Extensions: ext, Destination: filepath.Join(s.Destination, p.name)})
	}
	return s
}
func strictJSON(raw []byte, v any) error {
	if len(raw) == 0 || len(raw) > 1024*1024 {
		return errors.New("JSON document must contain 1..1048576 bytes")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return errors.New("JSON must contain exactly one document")
	}
	return nil
}
func DecodeSettings(raw []byte) (Settings, error) {
	var s Settings
	if err := strictJSON(raw, &s); err != nil {
		return s, err
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return s, err
	}
	for _, key := range []string{"schema", "autoSort", "intervalSeconds", "delayEnabled", "source", "destination", "historyDays", "categories", "rules", "ignore", "unknownFiles"} {
		value, ok := keys[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return s, fmt.Errorf("settings requires non-null field %q", key)
		}
	}

	for _, array := range []struct {
		key    string
		fields []string
	}{{"categories", []string{"id", "name", "enabled", "extensions", "destination"}}, {"rules", []string{"id", "name", "enabled", "pattern", "category"}}} {
		var entries []map[string]json.RawMessage
		if err := json.Unmarshal(keys[array.key], &entries); err != nil {
			return s, err
		}
		for i, entry := range entries {
			for _, key := range array.fields {
				value, ok := entry[key]
				if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
					return s, fmt.Errorf("%s[%d] requires non-null field %q", array.key, i, key)
				}
			}
		}
	}
	return s, nil
}
func (e *Engine) Settings() (Settings, error) {
	raw, err := os.ReadFile(e.configPath())
	if err != nil {
		return Settings{}, fmt.Errorf("read configuration: %w", err)
	}
	s, err := DecodeSettings(raw)
	if err != nil {
		return s, err
	}
	return e.validate(s)
}
func (e *Engine) validate(s Settings) (Settings, error) {
	if s.Schema != 1 {
		return s, fmt.Errorf("unsupported settings schema %d", s.Schema)
	}
	if s.IntervalSeconds < 60 || s.IntervalSeconds > 604800 {
		return s, fmt.Errorf("intervalSeconds %d must be an integer from 60 to 604800", s.IntervalSeconds)
	}
	if s.HistoryDays < 1 || s.HistoryDays > 365 {
		return s, fmt.Errorf("historyDays %d must be from 1 to 365", s.HistoryDays)
	}
	if s.UnknownFiles != "others" && s.UnknownFiles != "leave" {
		return s, fmt.Errorf("invalid unknownFiles %q", s.UnknownFiles)
	}
	if len(s.Categories) < 1 || len(s.Categories) > 100 || len(s.Rules) > 200 || len(s.Ignore) > 200 {
		return s, errors.New("limits: 1..100 categories, 200 rules and 200 ignore patterns")
	}
	var err error
	s.Source, err = e.validatePath(s.Source)
	if err != nil {
		return s, fmt.Errorf("source: %w", err)
	}
	s.Destination, err = e.validatePath(s.Destination)
	if err != nil {
		return s, fmt.Errorf("destination: %w", err)
	}
	if overlaps(s.Source, s.Destination) {
		return s, errors.New("source and destination must not overlap")
	}
	validID := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	seen := map[string]bool{}
	for i, c := range s.Categories {
		if !validID.MatchString(c.ID) || seen[c.ID] {
			return s, fmt.Errorf("invalid or duplicate category id %q", c.ID)
		}
		seen[c.ID] = true
		if strings.TrimSpace(c.Name) == "" || len(c.Name) > 100 {
			return s, fmt.Errorf("invalid category name %q", c.Name)
		}
		if c.Destination == "" {
			return s, fmt.Errorf("category %q needs an absolute destination", c.ID)
		}
		c.Destination, err = e.validatePath(c.Destination)
		if err != nil {
			return s, fmt.Errorf("category %q destination: %w", c.ID, err)
		}
		if overlaps(s.Source, c.Destination) {
			return s, fmt.Errorf("category %q destination overlaps source", c.ID)
		}
		if len(c.Extensions) > 300 {
			return s, fmt.Errorf("category %q has over 300 extensions", c.ID)
		}
		for j, x := range c.Extensions {
			if !regexp.MustCompile(`^[a-zA-Z0-9]+(?:\.[a-zA-Z0-9]+)*$`).MatchString(x) || len(x) > 40 {
				return s, fmt.Errorf("category %q has invalid extension %q", c.ID, x)
			}
			c.Extensions[j] = strings.ToLower(x)
		}
		s.Categories[i] = c
	}
	if !seen["others"] {
		return s, errors.New("the others category is required")
	}
	seenRules := map[string]bool{}
	for _, r := range s.Rules {
		if !validID.MatchString(r.ID) || seenRules[r.ID] || !seen[r.Category] {
			return s, fmt.Errorf("invalid rule %q or category %q", r.ID, r.Category)
		}
		seenRules[r.ID] = true
		if strings.TrimSpace(r.Name) == "" || len(r.Name) > 100 {
			return s, fmt.Errorf("invalid rule name %q", r.Name)
		}
		if err := validGlob(r.Pattern); err != nil {
			return s, fmt.Errorf("rule %q: %w", r.ID, err)
		}
	}
	for _, p := range s.Ignore {
		if err := validGlob(p); err != nil {
			return s, fmt.Errorf("ignore: %w", err)
		}
	}
	if s.Ignore == nil {
		s.Ignore = []string{}
	}
	if s.Rules == nil {
		s.Rules = []Rule{}
	}
	return s, nil
}
func validGlob(p string) error {
	if p == "" || len(p) > 256 || strings.ContainsAny(p, "/\x00") {
		return fmt.Errorf("invalid filename glob %q", p)
	}
	_, err := filepath.Match(p, "test")
	if err != nil {
		return fmt.Errorf("invalid filename glob %q: %w", p, err)
	}
	return nil
}
func (e *Engine) validatePath(p string) (string, error) {
	if p == "" || !filepath.IsAbs(p) || strings.IndexByte(p, 0) >= 0 {
		return "", fmt.Errorf("expected absolute shared-storage path, got %q", p)
	}
	// Android's public aliases point at the same primary shared-storage root.
	if e.Storage != "/sdcard" && (p == "/sdcard" || strings.HasPrefix(p, "/sdcard/")) {
		if actual, err := canonical("/sdcard"); err == nil && actual == e.Storage {
			p = filepath.Join(e.Storage, strings.TrimPrefix(p, "/sdcard"))
		}
	}
	result, err := canonical(p)
	if err != nil {
		return "", err
	}
	if !within(e.Storage, result) || result == e.Storage {
		return "", fmt.Errorf("path %q must be inside %q", p, e.Storage)
	}
	if overlaps(result, e.State) || overlaps(result, e.Module) {
		return "", fmt.Errorf("path %q overlaps protected Sortify state/module", p)
	}
	return result, nil
}
func canonical(p string) (string, error) {
	p = filepath.Clean(p)
	parent := p
	var suffix []string
	for {
		v, err := filepath.EvalSymlinks(parent)
		if err == nil {
			for i := len(suffix) - 1; i >= 0; i-- {
				v = filepath.Join(v, suffix[i])
			}
			return v, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("resolve %q: %w", p, err)
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", err
		}
		suffix = append(suffix, filepath.Base(parent))
		parent = next
	}
}
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func overlaps(a, b string) bool { return within(a, b) || within(b, a) }
func (e *Engine) Init(legacy int) (Settings, error) {
	if err := os.MkdirAll(e.State, 0700); err != nil {
		return Settings{}, err
	}
	lock, err := e.acquire()
	if err != nil {
		return Settings{}, err
	}
	defer lock.Close()
	if _, err := os.Stat(e.configPath()); err == nil {
		return e.Settings()
	} else if !errors.Is(err, os.ErrNotExist) {
		return Settings{}, err
	}
	s := e.Defaults()
	if legacy != 0 {
		if legacy < 60 || legacy > 604800 {
			return s, fmt.Errorf("legacy interval %d is outside 60..604800", legacy)
		}
		s.IntervalSeconds = legacy
		s.AutoSort = true
	}
	s, err = e.validate(s)
	if err != nil {
		return s, err
	}
	if err = atomicJSON(e.configPath(), s); err != nil {
		return s, err
	}
	db := emptyDB()
	if legacy != 0 {
		db.Warnings = append(db.Warnings, "Migrated v4 automatic sorting and interval. Keep recent downloads for 1 day is now enabled; review settings.")
	}
	if _, err := os.Stat(e.dbPath()); errors.Is(err, os.ErrNotExist) {
		err = e.saveDB(db)
		return s, err
	}
	return s, nil
}
func (e *Engine) SaveSettings(s Settings) error {
	validated, err := e.validate(s)
	if err != nil {
		return err
	}
	lock, err := acquirePath(filepath.Join(e.State, "config.lock"))
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = e.checkAccess(validated); err != nil {
		return err
	}
	return atomicJSON(e.configPath(), validated)
}
func (e *Engine) checkAccess(s Settings) error {
	// Open existing ancestors without following symlinks; do not create folders on an invalid save.
	paths := []string{s.Source, s.Destination}
	for _, c := range s.Categories {
		paths = append(paths, c.Destination)
	}
	for _, p := range paths {
		ancestor := p
		for {
			if _, err := os.Lstat(ancestor); err == nil {
				break
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("folder %q: %w", p, err)
			}
			ancestor = filepath.Dir(ancestor)
		}
		f, err := secureOpen(ancestor, os.O_RDONLY|directoryFlag, 0)
		if err != nil {
			return fmt.Errorf("folder %q: %w", p, err)
		}
		f.Close()
		if err := accessWritable(ancestor); err != nil {
			return fmt.Errorf("folder %q is not writable: %w", p, err)
		}
	}
	return nil
}
func warnings(s Settings) []string {
	out := []string{}
	for i, a := range s.Rules {
		if !a.Enabled {
			continue
		}
		for _, b := range s.Rules[i+1:] {
			if !b.Enabled {
				continue
			}
			ap, bp := strings.ToLower(a.Pattern), strings.ToLower(b.Pattern)
			ab, _ := filepath.Match(ap, bp)
			ba, _ := filepath.Match(bp, ap)
			if ap == bp || ap == "*" || bp == "*" || ab || ba {
				out = append(out, fmt.Sprintf("Rules %q and %q overlap; %q wins when both match.", a.Name, b.Name, a.Name))
			}
		}
	}

	mappings := map[string]string{}
	for _, c := range s.Categories {
		if !c.Enabled {
			continue
		}
		for _, ext := range c.Extensions {
			if first, ok := mappings[ext]; ok && first != c.Name {
				out = append(out, fmt.Sprintf("Extension .%s maps to %q and %q; %q wins unless a custom rule overrides it.", ext, first, c.Name, first))
			} else {
				mappings[ext] = c.Name
			}
		}
	}
	return out
}
