// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"runtime/internal/atomic"
	_ "unsafe" // for go:linkname
)

//go:generate go run wincallback.go
//go:generate go run mkduff.go
//go:generate go run mkfastlog2table.go

var ticks struct {
	lock mutex
	pad  uint32 // ensure 8-byte alignment of val on 386
	val  uint64
}

// Note: Called by runtime/pprof in addition to runtime code.
func tickspersecond() int64 {
	/// 每秒tick的值
	r := int64(atomic.Load64(&ticks.val))
	if r != 0 {
		return r
	}


	lock(&ticks.lock)
	r = int64(ticks.val)
	//// r等于0
	if r == 0 {
		t0 := nanotime() /// 纳秒
		c0 := cputicks() /// cpu tick
		usleep(100 * 1000)
		t1 := nanotime()
		c1 := cputicks()
		if t1 == t0 {
			t1++
		}
		r = (c1 - c0) * 1000 * 1000 * 1000 / (t1 - t0) /// 计算一个值
		if r == 0 {
			r++
		}
		atomic.Store64(&ticks.val, uint64(r))
	}

	unlock(&ticks.lock)
	return r
}

var envs []string
var argslice []string

//go:linkname syscall_runtime_envs syscall.runtime_envs
func syscall_runtime_envs() []string { return append([]string{}, envs...) }

//go:linkname syscall_Getpagesize syscall.Getpagesize
func syscall_Getpagesize() int { return int(physPageSize) }

//go:linkname os_runtime_args os.runtime_args
func os_runtime_args() []string { return append([]string{}, argslice...) }

//go:linkname syscall_Exit syscall.Exit
//go:nosplit
func syscall_Exit(code int) {
	exit(int32(code))
}
