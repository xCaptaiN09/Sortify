package engine

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

const (
	directoryFlag = syscall.O_DIRECTORY
	// Linux O_PATH is not exported by syscall on every supported architecture.
	// Parent handles need search permission, not permission to list their contents.
	pathFlag = 1 << 21
)

var ErrBusy = errors.New("Already running")

func id() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("system randomness unavailable: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}
func secureOpen(p string, flags int, mode uint32) (*os.File, error) {
	if !filepath.IsAbs(p) {
		return nil, fmt.Errorf("path is not absolute: %q", p)
	}
	clean := filepath.Clean(p)
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	rootFlags := pathFlag | directoryFlag
	if clean == "/" {
		rootFlags = flags
	}
	fd, err := syscall.Open("/", rootFlags|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, mode)
	if err != nil {
		return nil, fmt.Errorf("open root while accessing %q: %w", p, err)
	}
	component := "/"
	for i, part := range parts {
		if part == "" {
			continue
		}
		component = filepath.Join(component, part)
		f := pathFlag | directoryFlag
		if i == len(parts)-1 {
			f = flags
		}
		next, err := syscall.Openat(fd, part, f|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, mode)
		syscall.Close(fd)
		if err != nil {
			return nil, fmt.Errorf("open %q while accessing %q: %w", component, p, err)
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), p), nil
}
func accessWritable(p string) error { return syscall.Access(p, 2|1) }
func secureMkdir(p string) error {
	if !filepath.IsAbs(p) {
		return fmt.Errorf("folder path is not absolute: %q", p)
	}
	// Pin parents without listing them; O_NOFOLLOW still rejects replaced symlinks.
	fd, err := syscall.Open("/", pathFlag|directoryFlag|syscall.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { syscall.Close(fd) }()
	component := "/"
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Clean(p), "/"), "/") {
		if part == "" {
			continue
		}
		component = filepath.Join(component, part)
		next, err := syscall.Openat(fd, part, pathFlag|directoryFlag|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if errors.Is(err, syscall.ENOENT) {
			if err = syscall.Mkdirat(fd, part, 0755); err != nil && !errors.Is(err, syscall.EEXIST) {
				return fmt.Errorf("create %q while creating folder %q: %w", component, p, err)
			}
			next, err = syscall.Openat(fd, part, pathFlag|directoryFlag|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		}
		if err != nil {
			return fmt.Errorf("open %q while creating folder %q: %w", component, p, err)
		}
		syscall.Close(fd)
		fd = next
	}
	return nil
}
func syncDir(p string) error {
	f, err := secureOpen(p, os.O_RDONLY|directoryFlag, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	err = f.Sync()
	if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) {
		return nil
	}
	return err
}
func atomicJSON(p string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp, err := os.OpenFile(filepath.Join(filepath.Dir(p), ".write-"+id()), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("write state %q: %w", p, err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write state %q: %w", p, err)
	}
	if err = os.Rename(name, p); err != nil {
		return err
	}
	return syncDir(filepath.Dir(p))
}
func (e *Engine) dbPath() string { return filepath.Join(e.State, "state.json") }
func emptyDB() *database {
	return &database{History: []Run{}, Holds: []Hold{}, Observations: map[string]Observation{}, Warnings: []string{}}
}
func (e *Engine) loadDB() (*database, error) {
	data, err := os.ReadFile(e.dbPath())
	if errors.Is(err, os.ErrNotExist) {
		return emptyDB(), nil
	}
	if err != nil {
		return nil, err
	}
	db := emptyDB()
	if err = json.Unmarshal(data, db); err != nil {
		return nil, fmt.Errorf("state is corrupt; refusing file operations: %w", err)
	}
	if db.Observations == nil {
		db.Observations = map[string]Observation{}
	}
	return db, nil
}
func (e *Engine) saveDB(db *database) error { return atomicJSON(e.dbPath(), db) }

type Lock struct{ File *os.File }

func (l *Lock) Close() error {
	if l == nil || l.File == nil {
		return nil
	}
	return l.File.Close()
}
func (e *Engine) lockPath() string        { return filepath.Join(e.State, "operation.lock") }
func (e *Engine) acquire() (*Lock, error) { return acquirePath(e.lockPath()) }
func acquirePath(p string) (*Lock, error) {
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, err
	}
	return &Lock{File: f}, nil
}
func (e *Engine) InheritedLock(fd uintptr) (*Lock, error) {
	f := os.NewFile(fd, "operation.lock")
	if f == nil {
		return nil, errors.New("missing inherited operation lock")
	}
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	expected, err := os.Stat(e.lockPath())
	if err != nil || !os.SameFile(st, expected) {
		f.Close()
		return nil, errors.New("invalid inherited operation lock")
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, err
	}
	return &Lock{File: f}, nil
}
func identityInfo(st os.FileInfo) (Identity, error) {
	if !st.Mode().IsRegular() {
		return Identity{}, fmt.Errorf("not a regular file: %q", st.Name())
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return Identity{}, errors.New("file identity is unavailable")
	}
	return Identity{Device: uint64(sys.Dev), Inode: uint64(sys.Ino), Size: st.Size(), Modified: st.ModTime().UnixNano(), Changed: int64(sys.Ctim.Sec)*1e9 + int64(sys.Ctim.Nsec)}, nil
}
func statIdentity(p string) (Identity, error) {
	f, err := secureOpen(p, os.O_RDONLY, 0)
	if err != nil {
		return Identity{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Identity{}, err
	}
	return identityInfo(st)
}
func sameIdentity(a, b Identity) bool {
	return a.Device == b.Device && a.Inode == b.Inode && a.Size == b.Size && a.Modified == b.Modified && a.Changed == b.Changed
}
func sameContent(a, b Identity) bool { return a.Size == b.Size && a.Hash != "" && a.Hash == b.Hash }
func fingerprint(p string) (Identity, error) {
	f, err := secureOpen(p, os.O_RDONLY, 0)
	if err != nil {
		return Identity{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Identity{}, err
	}
	before, err := identityInfo(st)
	if err != nil {
		return Identity{}, err
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return Identity{}, err
	}
	st, err = f.Stat()
	if err != nil {
		return Identity{}, err
	}
	after, err := identityInfo(st)
	if err != nil {
		return Identity{}, err
	}
	pathNow, err := statIdentity(p)
	if err != nil {
		return Identity{}, err
	}
	if !sameIdentity(before, after) || !sameIdentity(after, pathNow) {
		return Identity{}, errors.New("file changed during content comparison")
	}
	after.Hash = hex.EncodeToString(h.Sum(nil))
	return after, nil
}
func equalBytes(a, b string) (bool, error) {
	fa, err := secureOpen(a, os.O_RDONLY, 0)
	if err != nil {
		return false, err
	}
	defer fa.Close()
	fb, err := secureOpen(b, os.O_RDONLY, 0)
	if err != nil {
		return false, err
	}
	defer fb.Close()
	ba := make([]byte, 128*1024)
	bb := make([]byte, len(ba))
	for {
		na, ea := io.ReadFull(fa, ba)
		nb, eb := io.ReadFull(fb, bb)
		if na != nb {
			return false, nil
		}
		for i := 0; i < na; i++ {
			if ba[i] != bb[i] {
				return false, nil
			}
		}
		if ea == io.EOF || ea == io.ErrUnexpectedEOF {
			return eb == ea, nil
		}
		if ea != nil {
			return false, ea
		}
		if eb != nil {
			return false, eb
		}
	}
}
func freeName(dir, name string, reserved map[string]bool) (string, bool, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		entries = nil
	} else if err != nil {
		return "", false, err
	}
	used := map[string]bool{}
	for _, v := range entries {
		used[strings.ToLower(v.Name())] = true
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for n := 0; n < 100000; n++ {
		candidate := name
		if n > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", base, n, ext)
		}
		p := filepath.Join(dir, candidate)
		if !used[strings.ToLower(candidate)] && !reserved[strings.ToLower(p)] {
			return p, n > 0, nil
		}
	}
	return "", false, errors.New("destination has too many naming collisions")
}

func privateStage(parent string) (string, *os.File, error) {
	f, err := secureOpen(parent, os.O_RDONLY|directoryFlag, 0)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	name := ".sortify-" + id()
	if err = syscall.Mkdirat(int(f.Fd()), name, 0700); err != nil {
		return "", nil, err
	}
	fd, err := syscall.Openat(int(f.Fd()), name, syscall.O_RDONLY|directoryFlag|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return "", nil, err
	}
	p := filepath.Join(parent, name)
	return p, os.NewFile(uintptr(fd), p), nil
}
func capturePath(source string, stage *os.File) error {
	parent, err := secureOpen(filepath.Dir(source), os.O_RDONLY|directoryFlag, 0)
	if err != nil {
		return err
	}
	defer parent.Close()
	// renameat2(RENAME_NOREPLACE) is required: emulated storage may not
	// enforce private directory modes. Unsupported filesystems fail closed.
	var trap uintptr
	switch runtime.GOARCH {
	case "amd64":
		trap = 316
	case "386":
		trap = 353
	case "arm":
		trap = 382
	case "arm64":
		trap = 276
	default:
		return fmt.Errorf("unsupported capture architecture %q", runtime.GOARCH)
	}
	oldName, err := syscall.BytePtrFromString(filepath.Base(source))
	if err != nil {
		return err
	}
	newName, err := syscall.BytePtrFromString("source")
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(trap, parent.Fd(), uintptr(unsafe.Pointer(oldName)), stage.Fd(), uintptr(unsafe.Pointer(newName)), 1, 0)
	if errno != 0 {
		return fmt.Errorf("capture without replacement: %w", errno)
	}
	return nil
}

func guardedRemove(directory *os.File, name string, expected Identity) error {
	if directory == nil || name == "" || filepath.Base(name) != name {
		return errors.New("invalid bound-directory removal")
	}
	fd, err := syscall.Openat(int(directory.Fd()), name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	child := os.NewFile(uintptr(fd), name)
	defer child.Close()
	info, err := child.Stat()
	if err != nil {
		return err
	}
	actual, err := identityInfo(info)
	if err != nil {
		return err
	}
	if !sameIdentity(actual, expected) {
		return errors.New("bound source identity changed; file preserved")
	}
	return syscall.Unlinkat(int(directory.Fd()), name)
}
