// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import _ "unsafe" // for go:linkname

//go:linkname setMaxStack runtime/debug.setMaxStack
func setMaxStack(in int) (out int) {
	out = int(maxstacksize)
	maxstacksize = uintptr(in) /// 设置栈的最大值；
	return out
}

//go:linkname setPanicOnFault runtime/debug.setPanicOnFault
func setPanicOnFault(new bool) (old bool) { /// 是否panic 在 fault的时候
	_g_ := getg()
	old = _g_.paniconfault
	_g_.paniconfault = new
	return old
}
