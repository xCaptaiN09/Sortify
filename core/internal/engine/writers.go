package engine

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// knownWriter is an additional signal, not a download-completion authority.
// Android SELinux and procfs hidepid can restrict other processes. Inaccessible
// processes are not claimed as checked; independent observations and repeated
// identity/content checks remain mandatory even when this reports no writer.
func knownWriter(identity Identity) bool {
	processes, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	major := (identity.Device>>8)&0xfff | (identity.Device>>32)&0xfffff000
	minor := identity.Device&0xff | (identity.Device>>12)&0xffffff00
	for _, process := range processes {
		if _, err := strconv.ParseUint(process.Name(), 10, 32); err != nil {
			continue
		}
		base := filepath.Join("/proc", process.Name())
		descriptors, err := os.ReadDir(filepath.Join(base, "fd"))
		if err == nil {
			for _, descriptor := range descriptors {
				info, err := os.Stat(filepath.Join(base, "fd", descriptor.Name()))
				if err != nil {
					continue
				}
				st, ok := info.Sys().(*syscall.Stat_t)
				if !ok || uint64(st.Dev) != identity.Device || uint64(st.Ino) != identity.Inode {
					continue
				}
				raw, err := os.ReadFile(filepath.Join(base, "fdinfo", descriptor.Name()))
				if err != nil {
					continue
				}
				for _, line := range strings.Split(string(raw), "\n") {
					fields := strings.Fields(line)
					if len(fields) == 2 && fields[0] == "flags:" {
						flags, err := strconv.ParseUint(fields[1], 8, 64)
						if err == nil && flags&syscall.O_ACCMODE != syscall.O_RDONLY {
							return true
						}
					}
				}
			}
		}
		maps, err := os.ReadFile(filepath.Join(base, "maps"))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(maps), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 5 || len(fields[1]) < 4 || fields[1][1] != 'w' || fields[1][3] != 's' {
				continue
			}
			inode, err := strconv.ParseUint(fields[4], 10, 64)
			if err != nil || inode != identity.Inode {
				continue
			}
			device := strings.Split(fields[3], ":")
			if len(device) != 2 {
				continue
			}
			a, aerr := strconv.ParseUint(device[0], 16, 64)
			b, berr := strconv.ParseUint(device[1], 16, 64)
			if aerr == nil && berr == nil && a == major && b == minor {
				return true
			}
		}
	}
	return false
}
