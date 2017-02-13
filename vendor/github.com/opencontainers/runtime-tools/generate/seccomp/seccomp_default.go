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
func DefaultProfile(rs *specs.Spec) *rspec.Seccomp {

	syscalls := []rspec.Syscall{
		{
			Name:   "accept",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "accept4",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "access",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "alarm",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "bind",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "brk",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "capget",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "capset",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "chdir",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "chmod",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "chown",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "chown32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},

		{
			Name:   "clock_getres",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "clock_gettime",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "clock_nanosleep",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "close",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "connect",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "copy_file_range",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "creat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "dup",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "dup2",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "dup3",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "epoll_create",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "epoll_create1",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "epoll_ctl",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "epoll_ctl_old",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "epoll_pwait",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "epoll_wait",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "epoll_wait_old",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "eventfd",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "eventfd2",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "execve",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "execveat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "exit",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "exit_group",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "faccessat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fadvise64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fadvise64_64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fallocate",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fanotify_mark",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fchdir",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fchmod",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fchmodat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fchown",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fchown32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fchownat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fcntl",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fcntl64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fdatasync",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fgetxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "flistxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "flock",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fork",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fremovexattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fsetxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fstat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fstat64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fstatat64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fstatfs",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fstatfs64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "fsync",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "ftruncate",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "ftruncate64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "futex",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "futimesat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getcpu",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getcwd",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getdents",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getdents64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getegid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getegid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "geteuid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "geteuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getgid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getgroups",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getgroups32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getitimer",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getpeername",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getpgid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getpgrp",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getpid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getppid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getpriority",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getrandom",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getresgid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getresgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getresuid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getresuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getrlimit",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "get_robust_list",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getrusage",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getsid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getsockname",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getsockopt",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "get_thread_area",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "gettid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "gettimeofday",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getuid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "getxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "inotify_add_watch",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "inotify_init",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "inotify_init1",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "inotify_rm_watch",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "io_cancel",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "ioctl",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "io_destroy",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "io_getevents",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "ioprio_get",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "ioprio_set",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "io_setup",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "io_submit",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "ipc",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "kill",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "lchown",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "lchown32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "lgetxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "link",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "linkat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "listen",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "listxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "llistxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "_llseek",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "lremovexattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "lseek",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "lsetxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "lstat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "lstat64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "madvise",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "memfd_create",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mincore",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mkdir",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mkdirat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mknod",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mknodat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mlock",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mlock2",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mlockall",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mmap",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mmap2",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mprotect",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mq_getsetattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mq_notify",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mq_open",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mq_timedreceive",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mq_timedsend",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mq_unlink",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "mremap",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "msgctl",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "msgget",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "msgrcv",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "msgsnd",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "msync",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "munlock",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "munlockall",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "munmap",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "nanosleep",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "newfstatat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "_newselect",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "open",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "openat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "pause",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "personality",
			Action: rspec.ActAllow,
			Args: []rspec.Arg{
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
			Args: []rspec.Arg{
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
			Args: []rspec.Arg{
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
			Args:   []rspec.Arg{},
		},
		{
			Name:   "pipe2",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "poll",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "ppoll",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "prctl",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "pread64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "preadv",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "prlimit64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "pselect6",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "pwrite64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "pwritev",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "read",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "readahead",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "readlink",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "readlinkat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "readv",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "recv",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "recvfrom",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "recvmmsg",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "recvmsg",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "remap_file_pages",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "removexattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rename",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "renameat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "renameat2",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "restart_syscall",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rmdir",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rt_sigaction",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rt_sigpending",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rt_sigprocmask",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rt_sigqueueinfo",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rt_sigreturn",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rt_sigsuspend",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rt_sigtimedwait",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "rt_tgsigqueueinfo",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_getaffinity",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_getattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_getparam",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_get_priority_max",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_get_priority_min",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_getscheduler",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_rr_get_interval",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_setaffinity",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_setattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_setparam",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_setscheduler",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sched_yield",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "seccomp",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "select",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "semctl",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "semget",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "semop",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "semtimedop",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "send",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sendfile",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sendfile64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sendmmsg",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sendmsg",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sendto",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setfsgid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setfsgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setfsuid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setfsuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setgid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setgroups",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setgroups32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setitimer",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setpgid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setpriority",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setregid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setregid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setresgid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setresgid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setresuid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setresuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setreuid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setreuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setrlimit",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "set_robust_list",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setsid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setsockopt",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "set_thread_area",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "set_tid_address",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setuid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setuid32",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "setxattr",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "shmat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "shmctl",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "shmdt",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "shmget",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "shutdown",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sigaltstack",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "signalfd",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "signalfd4",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sigreturn",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "socket",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "socketcall",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "socketpair",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "splice",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "stat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "stat64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "statfs",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "statfs64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "symlink",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "symlinkat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sync",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sync_file_range",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "syncfs",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "sysinfo",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "syslog",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "tee",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "tgkill",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "time",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "timer_create",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "timer_delete",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "timerfd_create",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "timerfd_gettime",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "timerfd_settime",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "timer_getoverrun",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "timer_gettime",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "timer_settime",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "times",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "tkill",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "truncate",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "truncate64",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "ugetrlimit",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "umask",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "uname",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "unlink",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "unlinkat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "utime",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "utimensat",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "utimes",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "vfork",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "vmsplice",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "wait4",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "waitid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "waitpid",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "write",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
		{
			Name:   "writev",
			Action: rspec.ActAllow,
			Args:   []rspec.Arg{},
		},
	}
	var sysCloneFlagsIndex uint

	capSysAdmin := false
	var cap string

	for _, cap = range rs.Process.Capabilities {
		switch cap {
		case "CAP_DAC_READ_SEARCH":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "open_by_handle_at",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_ADMIN":
			capSysAdmin = true
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "bpf",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "clone",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "fanotify_init",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "lookup_dcookie",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "mount",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "name_to_handle_at",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "perf_event_open",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "setdomainname",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "sethostname",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "setns",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "umount",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "umount2",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "unshare",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_BOOT":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "reboot",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_CHROOT":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "chroot",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_MODULE":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "delete_module",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "init_module",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "finit_module",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "query_module",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_PACCT":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "acct",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_PTRACE":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "kcmp",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "process_vm_readv",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "process_vm_writev",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "ptrace",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_RAWIO":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "iopl",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "ioperm",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_TIME":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "settimeofday",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "stime",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
				{
					Name:   "adjtimex",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		case "CAP_SYS_TTY_CONFIG":
			syscalls = append(syscalls, []rspec.Syscall{
				{
					Name:   "vhangup",
					Action: rspec.ActAllow,
					Args:   []rspec.Arg{},
				},
			}...)
		}
	}

	if !capSysAdmin {
		syscalls = append(syscalls, []rspec.Syscall{
			{
				Name:   "clone",
				Action: rspec.ActAllow,
				Args: []rspec.Arg{
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
		syscalls = append(syscalls, []rspec.Syscall{
			{
				Name:   "breakpoint",
				Action: rspec.ActAllow,
				Args:   []rspec.Arg{},
			},
			{
				Name:   "cacheflush",
				Action: rspec.ActAllow,
				Args:   []rspec.Arg{},
			},
			{
				Name:   "set_tls",
				Action: rspec.ActAllow,
				Args:   []rspec.Arg{},
			},
		}...)
	case "amd64", "x32":
		syscalls = append(syscalls, []rspec.Syscall{
			{
				Name:   "arch_prctl",
				Action: rspec.ActAllow,
				Args:   []rspec.Arg{},
			},
		}...)
		fallthrough
	case "x86":
		syscalls = append(syscalls, []rspec.Syscall{
			{
				Name:   "modify_ldt",
				Action: rspec.ActAllow,
				Args:   []rspec.Arg{},
			},
		}...)
	case "s390", "s390x":
		syscalls = append(syscalls, []rspec.Syscall{
			{
				Name:   "s390_pci_mmio_read",
				Action: rspec.ActAllow,
				Args:   []rspec.Arg{},
			},
			{
				Name:   "s390_pci_mmio_write",
				Action: rspec.ActAllow,
				Args:   []rspec.Arg{},
			},
			{
				Name:   "s390_runtime_instr",
				Action: rspec.ActAllow,
				Args:   []rspec.Arg{},
			},
		}...)
		/* Flags parameter of the clone syscall is the 2nd on s390 */
	}

	return &rspec.Seccomp{
		DefaultAction: rspec.ActErrno,
		Architectures: arches(),
		Syscalls:      syscalls,
	}
}
