package client

import (
	"syscall"
	"unsafe"
)

// processStartTimeNano returns the process start time in nanoseconds since the
// Unix epoch for the given PID, using the macOS sysctl KERN_PROC interface.
// Returns 0 if the PID is not found or the sysctl call fails.
func processStartTimeNano(pid int) int64 {
	// struct kinfo_proc is large; we only need the start time embedded in it.
	// Use KERN_PROC / KERN_PROC_PID to retrieve it.
	mib := []int32{1 /* CTL_KERN */, 14 /* KERN_PROC */, 1 /* KERN_PROC_PID */, int32(pid)}

	// First call: get required buffer size
	var sz uintptr
	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		0, uintptr(unsafe.Pointer(&sz)),
		0, 0,
	)
	if errno != 0 || sz == 0 {
		return 0
	}

	// Second call: fill the buffer
	buf := make([]byte, sz)
	_, _, errno = syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&sz)),
		0, 0,
	)
	if errno != 0 || sz == 0 {
		return 0
	}

	// kinfo_proc layout on Darwin/arm64:
	//   The p_starttime field (struct timeval) sits at a fixed offset inside
	//   extern_proc (first member of kinfo_proc).
	//   extern_proc offset 0 is p_un (8 bytes), then p_vmspace (8), p_sigacts (8),
	//   p_flag (4), p_stat (1), p_pid (4), ... p_starttime at offset 84 on arm64.
	//
	// Because the struct layout is stable across macOS versions (it is part of
	// the public BSD ABI), we read it directly. The offset below matches
	// <sys/sysctl.h> / <sys/proc.h> for Darwin arm64.
	//
	// struct timeval { __darwin_time_t tv_sec; __darwin_suseconds_t tv_usec; }
	// p_starttime is at byte offset 84 in extern_proc on arm64.
	const startTimeOffset = 84
	if int(sz) < startTimeOffset+16 {
		return 0
	}

	// tv_sec  — 8-byte little-endian int64
	tvSec := *(*int64)(unsafe.Pointer(&buf[startTimeOffset]))
	// tv_usec — 4-byte little-endian int32 (Darwin suseconds_t is int32 on arm64)
	tvUsec := *(*int32)(unsafe.Pointer(&buf[startTimeOffset+8]))

	return tvSec*1e9 + int64(tvUsec)*1e3
}
