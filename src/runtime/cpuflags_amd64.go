// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"internal/cpu"
)

var useAVXmemmove bool

func init() {
	// Let's remove stepping and reserved fields
	/// processorVersionInfo 是一个整数
	///

	processor := processorVersionInfo & 0x0FFF3FF0

	isIntelBridgeFamily := isIntel && /// 是否是桥接家庭模式???
		processor == 0x206A0 ||
		processor == 0x206D0 ||
		processor == 0x306A0 ||
		processor == 0x306E0

	useAVXmemmove = cpu.X86.HasAVX && !isIntelBridgeFamily /// 使用的AVX内存方法
}
