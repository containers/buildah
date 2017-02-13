package seccomp

import (
	"runtime"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
)

func arches() []rspec.Arch {
	native := runtime.GOARCH

	switch native {
	case "amd64":
		return []rspec.Arch{rspec.ArchX86_64, rspec.ArchX86, rspec.ArchX32}
	case "arm64":
		return []rspec.Arch{rspec.ArchARM, rspec.ArchAARCH64}
	case "mips64":
		return []rspec.Arch{rspec.ArchMIPS, rspec.ArchMIPS64, rspec.ArchMIPS64N32}
	case "mips64n32":
		return []rspec.Arch{rspec.ArchMIPS, rspec.ArchMIPS64, rspec.ArchMIPS64N32}
	case "mipsel64":
		return []rspec.Arch{rspec.ArchMIPSEL, rspec.ArchMIPSEL64, rspec.ArchMIPSEL64N32}
	case "mipsel64n32":
		return []rspec.Arch{rspec.ArchMIPSEL, rspec.ArchMIPSEL64, rspec.ArchMIPSEL64N32}
	case "s390x":
		return []rspec.Arch{rspec.ArchS390, rspec.ArchS390X}
	default:
		return []rspec.Arch{}
	}
}

// DefaultProfile defines the whitelist for the default seccomp profile.
func DefaultProfile(rs *specs.Spec) *rspec.LinuxSeccomp {

	syscalls := []rspec.LinuxSyscall{
		{
			Name:   "accept",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "accept4",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "access",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "alarm",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "bind",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "brk",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "capget",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "capset",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "chdir",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "chmod",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "chown",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "chown32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},

		{
			Name:   "clock_getres",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "clock_gettime",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "clock_nanosleep",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "close",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "connect",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "copy_file_range",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "creat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "dup",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "dup2",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "dup3",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "epoll_create",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "epoll_create1",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "epoll_ctl",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "epoll_ctl_old",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "epoll_pwait",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "epoll_wait",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "epoll_wait_old",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "eventfd",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "eventfd2",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "execve",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "execveat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "exit",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "exit_group",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "faccessat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fadvise64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fadvise64_64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fallocate",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fanotify_mark",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fchdir",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fchmod",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fchmodat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fchown",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fchown32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fchownat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fcntl",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fcntl64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fdatasync",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fgetxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "flistxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "flock",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fork",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fremovexattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fsetxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fstat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fstat64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fstatat64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fstatfs",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fstatfs64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "fsync",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "ftruncate",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "ftruncate64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "futex",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "futimesat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getcpu",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getcwd",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getdents",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getdents64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getegid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getegid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "geteuid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "geteuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getgid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getgroups",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getgroups32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getitimer",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getpeername",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getpgid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getpgrp",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getpid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getppid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getpriority",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getrandom",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getresgid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getresgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getresuid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getresuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getrlimit",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "get_robust_list",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getrusage",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getsid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getsockname",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getsockopt",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "get_thread_area",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "gettid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "gettimeofday",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getuid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "getxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "inotify_add_watch",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "inotify_init",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "inotify_init1",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "inotify_rm_watch",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "io_cancel",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "ioctl",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "io_destroy",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "io_getevents",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "ioprio_get",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "ioprio_set",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "io_setup",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "io_submit",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "ipc",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "kill",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "lchown",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "lchown32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "lgetxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "link",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "linkat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "listen",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "listxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "llistxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "_llseek",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "lremovexattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "lseek",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "lsetxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "lstat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "lstat64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "madvise",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "memfd_create",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mincore",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mkdir",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mkdirat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mknod",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mknodat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mlock",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mlock2",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mlockall",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mmap",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mmap2",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mprotect",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mq_getsetattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mq_notify",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mq_open",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mq_timedreceive",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mq_timedsend",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mq_unlink",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "mremap",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "msgctl",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "msgget",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "msgrcv",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "msgsnd",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "msync",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "munlock",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "munlockall",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "munmap",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "nanosleep",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "newfstatat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "_newselect",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "open",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "openat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "pause",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "personality",
			Action: rspec.ActAllow,
			Args: []rspec.LinuxSeccompArg{
				{
					Index: 0,
					Value: 0x0,
					Op:    rspec.OpEqualTo,
				},
			},
		},
		{
			Name:   "personality",
			Action: rspec.ActAllow,
			Args: []rspec.LinuxSeccompArg{
				{
					Index: 0,
					Value: 0x0008,
					Op:    rspec.OpEqualTo,
				},
			},
		},
		{
			Name:   "personality",
			Action: rspec.ActAllow,
			Args: []rspec.LinuxSeccompArg{
				{
					Index: 0,
					Value: 0xffffffff,
					Op:    rspec.OpEqualTo,
				},
			},
		},
		{
			Name:   "pipe",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "pipe2",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "poll",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "ppoll",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "prctl",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "pread64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "preadv",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "prlimit64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "pselect6",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "pwrite64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "pwritev",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "read",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "readahead",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "readlink",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "readlinkat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "readv",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "recv",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "recvfrom",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "recvmmsg",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "recvmsg",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "remap_file_pages",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "removexattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rename",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "renameat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "renameat2",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "restart_syscall",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rmdir",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rt_sigaction",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rt_sigpending",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rt_sigprocmask",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rt_sigqueueinfo",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rt_sigreturn",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rt_sigsuspend",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rt_sigtimedwait",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "rt_tgsigqueueinfo",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_getaffinity",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_getattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_getparam",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_get_priority_max",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_get_priority_min",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_getscheduler",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_rr_get_interval",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_setaffinity",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_setattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_setparam",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_setscheduler",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sched_yield",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "seccomp",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "select",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "semctl",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "semget",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "semop",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "semtimedop",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "send",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sendfile",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sendfile64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sendmmsg",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sendmsg",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sendto",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setfsgid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setfsgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setfsuid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setfsuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setgid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setgroups",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setgroups32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setitimer",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setpgid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setpriority",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setregid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setregid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setresgid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setresgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setresuid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setresuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setreuid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setreuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setrlimit",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "set_robust_list",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setsid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setsockopt",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "set_thread_area",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "set_tid_address",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setuid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "setxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "shmat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "shmctl",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "shmdt",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "shmget",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "shutdown",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sigaltstack",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "signalfd",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "signalfd4",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sigreturn",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "socket",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "socketcall",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "socketpair",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "splice",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "stat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "stat64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "statfs",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "statfs64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "symlink",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "symlinkat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sync",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sync_file_range",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "syncfs",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "sysinfo",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "syslog",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "tee",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "tgkill",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "time",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "timer_create",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "timer_delete",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "timerfd_create",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "timerfd_gettime",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "timerfd_settime",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "timer_getoverrun",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "timer_gettime",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "timer_settime",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "times",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "tkill",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "truncate",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "truncate64",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "ugetrlimit",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "umask",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "uname",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "unlink",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "unlinkat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "utime",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "utimensat",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "utimes",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "vfork",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "vmsplice",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "wait4",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "waitid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "waitpid",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "write",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
		{
			Name:   "writev",
			Action: rspec.ActAllow,
			Args:   []rspec.LinuxSeccompArg{},
		},
	}
	var sysCloneFlagsIndex uint

	capSysAdmin := false
	var cap string

	for _, cap = range rs.Process.Capabilities {
		switch cap {
		case "CAP_DAC_READ_SEARCH":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "open_by_handle_at",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_ADMIN":
			capSysAdmin = true
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "bpf",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "clone",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "fanotify_init",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "lookup_dcookie",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "mount",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "name_to_handle_at",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "perf_event_open",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "setdomainname",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "sethostname",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "setns",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "umount",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "umount2",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "unshare",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_BOOT":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "reboot",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_CHROOT":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "chroot",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_MODULE":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "delete_module",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "init_module",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "finit_module",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "query_module",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_PACCT":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "acct",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_PTRACE":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "kcmp",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "process_vm_readv",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "process_vm_writev",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "ptrace",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_RAWIO":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "iopl",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "ioperm",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_TIME":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "settimeofday",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "stime",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
				{
					Name:   "adjtimex",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		case "CAP_SYS_TTY_CONFIG":
			syscalls = append(syscalls, []rspec.LinuxSyscall{
				{
					Name:   "vhangup",
					Action: rspec.ActAllow,
					Args:   []rspec.LinuxSeccompArg{},
				},
			}...)
		}
	}

	if !capSysAdmin {
		syscalls = append(syscalls, []rspec.LinuxSyscall{
			{
				Name:   "clone",
				Action: rspec.ActAllow,
				Args: []rspec.LinuxSeccompArg{
					{
						Index:    sysCloneFlagsIndex,
						Value:    syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUSER | syscall.CLONE_NEWPID | syscall.CLONE_NEWNET,
						ValueTwo: 0,
						Op:       rspec.OpMaskedEqual,
					},
				},
			},
		}...)

	}

	arch := runtime.GOARCH
	switch arch {
	case "arm", "arm64":
		syscalls = append(syscalls, []rspec.LinuxSyscall{
			{
				Name:   "breakpoint",
				Action: rspec.ActAllow,
				Args:   []rspec.LinuxSeccompArg{},
			},
			{
				Name:   "cacheflush",
				Action: rspec.ActAllow,
				Args:   []rspec.LinuxSeccompArg{},
			},
			{
				Name:   "set_tls",
				Action: rspec.ActAllow,
				Args:   []rspec.LinuxSeccompArg{},
			},
		}...)
	case "amd64", "x32":
		syscalls = append(syscalls, []rspec.LinuxSyscall{
			{
				Name:   "arch_prctl",
				Action: rspec.ActAllow,
				Args:   []rspec.LinuxSeccompArg{},
			},
		}...)
		fallthrough
	case "x86":
		syscalls = append(syscalls, []rspec.LinuxSyscall{
			{
				Name:   "modify_ldt",
				Action: rspec.ActAllow,
				Args:   []rspec.LinuxSeccompArg{},
			},
		}...)
	case "s390", "s390x":
		syscalls = append(syscalls, []rspec.LinuxSyscall{
			{
				Name:   "s390_pci_mmio_read",
				Action: rspec.ActAllow,
				Args:   []rspec.LinuxSeccompArg{},
			},
			{
				Name:   "s390_pci_mmio_write",
				Action: rspec.ActAllow,
				Args:   []rspec.LinuxSeccompArg{},
			},
			{
				Name:   "s390_runtime_instr",
				Action: rspec.ActAllow,
				Args:   []rspec.LinuxSeccompArg{},
			},
		}...)
		/* Flags parameter of the clone syscall is the 2nd on s390 */
	}

	return &rspec.LinuxSeccomp{
		DefaultAction: rspec.ActErrno,
		Architectures: arches(),
		Syscalls:      syscalls,
	}
}
