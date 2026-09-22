//go:build linux && (amd64 || arm64)

package extract

import (
	"errors"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// The worker is a separate process. Restrict its current OS thread and exec on
// that same thread; exec discards every other Go runtime thread. Landlock ABI3
// is the minimum because ABI1/2 cannot deny file truncation.
func enterSandbox(directory, program string) error {
	runtime.LockOSThread()
	for resource, limit := range map[int]uint64{unix.RLIMIT_CPU: 45, unix.RLIMIT_AS: 2 << 30, unix.RLIMIT_FSIZE: 32 << 20, unix.RLIMIT_NOFILE: 64, unix.RLIMIT_NPROC: 256, unix.RLIMIT_CORE: 0} {
		if e := unix.Setrlimit(resource, &unix.Rlimit{Cur: limit, Max: limit}); e != nil {
			return e
		}
	}
	if e := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); e != nil {
		return e
	}
	if e := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); e != nil {
		return e
	}
	if e := landlockPaths(directory, program); e != nil {
		return e
	}
	if e := unix.Chdir(directory); e != nil {
		return e
	}
	return denyEscapeSyscalls()
}

func landlockPaths(directory, program string) error {
	abi, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if errno != 0 || abi < 3 {
		return errors.New("Landlock ABI 3 이상이 필요합니다")
	}
	// ABI3 handled set includes all filesystem access classes except ioctl,
	// which was introduced later. Do not grant device creation or symlinks.
	const all uint64 = (1 << 15) - 1
	attr := struct{ Handled uint64 }{all}
	fd, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if errno != 0 {
		return errno
	}
	defer unix.Close(int(fd))
	const read = unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_READ_DIR
	const execute = read | unix.LANDLOCK_ACCESS_FS_EXECUTE
	const write = read | unix.LANDLOCK_ACCESS_FS_WRITE_FILE | unix.LANDLOCK_ACCESS_FS_REMOVE_DIR | unix.LANDLOCK_ACCESS_FS_REMOVE_FILE | unix.LANDLOCK_ACCESS_FS_MAKE_DIR | unix.LANDLOCK_ACCESS_FS_MAKE_REG | unix.LANDLOCK_ACCESS_FS_REFER | unix.LANDLOCK_ACCESS_FS_TRUNCATE
	entries := []struct {
		path     string
		access   uint64
		required bool
	}{
		{directory, write, true}, {program, execute, true},
		{"/lib", execute, true}, {"/usr/lib", execute, true}, {"/lib64", execute, false},
		{"/etc/ld.so.cache", read, false}, {"/etc/ld.so.conf", read, false},
		{"/etc/fonts", read, false}, {"/usr/share/fonts", read, false}, {"/var/cache/fontconfig", read, false},
		{"/usr/share/poppler", read, false}, {"/usr/share/tessdata", read, false}, {"/usr/share/tesseract-ocr", read, false},
		{"/dev/null", unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_WRITE_FILE, false},
		{"/dev/urandom", unix.LANDLOCK_ACCESS_FS_READ_FILE, false},
	}
	for _, entry := range entries {
		pathFD, e := unix.Open(entry.path, unix.O_PATH|unix.O_CLOEXEC, 0)
		if e != nil {
			if !entry.required && os.IsNotExist(e) {
				continue
			}
			return e
		}
		var stat unix.Stat_t
		e = unix.Fstat(pathFD, &stat)
		allowed := entry.access
		if stat.Mode&unix.S_IFMT != unix.S_IFDIR {
			allowed &= unix.LANDLOCK_ACCESS_FS_EXECUTE | unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_WRITE_FILE | unix.LANDLOCK_ACCESS_FS_TRUNCATE
		}
		// Packed C structure is 8+4 bytes, no Go trailing alignment padding.
		attr := struct {
			Allowed uint64
			Parent  int32
		}{allowed, int32(pathFD)}
		if e == nil {
			_, _, errno = unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, fd, 1, uintptr(unsafe.Pointer(&attr)), 0, 0, 0)
			if errno != 0 {
				e = errno
			}
		}
		unix.Close(pathFD)
		if e != nil {
			return e
		}
	}
	_, _, errno = unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, fd, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func denyEscapeSyscalls() error {
	arch := uint32(unix.AUDIT_ARCH_X86_64)
	if runtime.GOARCH == "arm64" {
		arch = unix.AUDIT_ARCH_AARCH64
	}
	load := func(offset uint32) unix.SockFilter {
		return unix.SockFilter{Code: unix.BPF_LD | unix.BPF_W | unix.BPF_ABS, K: offset}
	}
	ret := func(value uint32) unix.SockFilter { return unix.SockFilter{Code: unix.BPF_RET | unix.BPF_K, K: value} }
	filter := []unix.SockFilter{load(4), {Code: unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K, K: arch, Jt: 1}, ret(unix.SECCOMP_RET_KILL_PROCESS), load(0)}
	// Reject the x32 syscall namespace as well as alternate audit architectures.
	if runtime.GOARCH == "amd64" {
		filter = append(filter, unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JGE | unix.BPF_K, K: 0x40000000, Jf: 1}, ret(unix.SECCOMP_RET_KILL_PROCESS))
	}
	// Only in-process library threads may be cloned; do not let a compromised
	// parser fork more address-space budgets. clone3 callers fall back to clone.
	filter = append(filter, unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K, K: unix.SYS_CLONE, Jf: 4}, load(16), unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JSET | unix.BPF_K, K: unix.CLONE_THREAD, Jt: 1}, ret(unix.SECCOMP_RET_ERRNO|uint32(unix.EPERM)), load(0))
	filter = append(filter, unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K, K: unix.SYS_CLONE3, Jf: 1}, ret(unix.SECCOMP_RET_ERRNO|uint32(unix.ENOSYS)))
	if runtime.GOARCH == "amd64" { // fork/vfork do not exist in the arm64 table.
		for _, call := range []uint32{57, 58} {
			filter = append(filter, unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K, K: call, Jf: 1}, ret(unix.SECCOMP_RET_ERRNO|uint32(unix.EPERM)))
		}
	}
	for _, call := range []uint32{unix.SYS_SOCKET, unix.SYS_SOCKETPAIR, unix.SYS_CONNECT, unix.SYS_BIND, unix.SYS_LISTEN, unix.SYS_ACCEPT, unix.SYS_ACCEPT4, unix.SYS_SENDTO, unix.SYS_SENDMSG, unix.SYS_SENDMMSG, unix.SYS_RECVFROM, unix.SYS_RECVMSG, unix.SYS_RECVMMSG, unix.SYS_PTRACE, unix.SYS_PROCESS_VM_READV, unix.SYS_PROCESS_VM_WRITEV, unix.SYS_PIDFD_GETFD, unix.SYS_PIDFD_SEND_SIGNAL, unix.SYS_KILL, unix.SYS_TKILL, unix.SYS_TGKILL, unix.SYS_IO_URING_SETUP, unix.SYS_IO_URING_ENTER, unix.SYS_IO_URING_REGISTER, unix.SYS_OPEN_BY_HANDLE_AT, unix.SYS_BPF, unix.SYS_PERF_EVENT_OPEN, unix.SYS_MOUNT, unix.SYS_UMOUNT2, unix.SYS_PIVOT_ROOT, unix.SYS_CHROOT, unix.SYS_SETNS, unix.SYS_UNSHARE, unix.SYS_KEYCTL, unix.SYS_ADD_KEY, unix.SYS_REQUEST_KEY} {
		filter = append(filter, unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K, K: call, Jf: 1}, ret(unix.SECCOMP_RET_ERRNO|uint32(unix.EPERM)))
	}
	filter = append(filter, ret(unix.SECCOMP_RET_ALLOW))
	prog := unix.SockFprog{Len: uint16(len(filter)), Filter: &filter[0]}
	_, _, errno := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, unix.SECCOMP_FILTER_FLAG_TSYNC, uintptr(unsafe.Pointer(&prog)))
	if errno != 0 {
		return errno
	}
	return nil
}

func replaceProcess(program string, args, env []string) error { return unix.Exec(program, args, env) }
