// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sys

const (
	ArchFamily          = AMD64
	BigEndian           = false   /// 小端
	DefaultPhysPageSize = 4096
	PCQuantum           = 1
	Int64Align          = 8
	MinFrameSize        = 0
)

type Uintreg uint64
