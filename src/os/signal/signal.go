// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package signal

import (
	"os"
	"sync"
)

var handlers struct {
	/// 互斥锁
	sync.Mutex

	// Map a channel to the signals that should be sent to it.
	/// 信号通过这个 channel 发送
	m map[chan<- os.Signal]*handler

	// Map a signal to the number of channels receiving it.
	/// 指定的信号的接收数
	ref [numSig]int64

	// Map channels to signals while the channel is being stopped.
	// Not a map because entries live here only very briefly.
	// We need a separate container because we need m to correspond to ref
	// at all times, and we also need to keep track of the *handler
	// value for a channel being stopped. See the Stop function.
	/// 当这个channel停止的时候，通知机制
	stopping []stopping
}

type stopping struct {
	c chan<- os.Signal
	h *handler
}

type handler struct {
	mask [(numSig + 31) / 32]uint32 /// (numSig + 31) / 32 == 3 ///# mask [3]uint32
}

/// 检测是否存在该信号
func (h *handler) want(sig int) bool {
	return (h.mask[sig/32]>>uint(sig&31))&1 != 0   /// 0-31 == 0; 32-65 == 1 ; 11110 &
}

/// 设置该信号位
func (h *handler) set(sig int) {
	h.mask[sig/32] |= 1 << uint(sig&31)  /// 增加
}

/// 清除该信号位
func (h *handler) clear(sig int) {
	h.mask[sig/32] &^= 1 << uint(sig&31) /// 减少
}

// Stop relaying the signals, sigs, to any channels previously registered to
// receive them and either reset the signal handlers to their original values
// (action=disableSignal) or ignore the signals (action=ignoreSignal).
func cancel(sigs []os.Signal, action func(int)) {
	handlers.Lock()
	defer handlers.Unlock()

	remove := func(n int) {
		var zerohandler handler

		for c, h := range handlers.m {
			// 是否存在n
			if h.want(n) {
				// 存在减少引用数
				handlers.ref[n]--
				// 清除标记位
				h.clear(n)

				// 如果等于空标记位; 则删除
				if h.mask == zerohandler.mask {
					delete(handlers.m, c)
				}
			}
		}

		action(n)
	}

	if len(sigs) == 0 {
		for n := 0; n < numSig; n++ {
			remove(n)
		}
	} else {
		for _, s := range sigs {
			remove(signum(s))
		}
	}
}

// Ignore causes the provided signals to be ignored. If they are received by
// the program, nothing will happen. Ignore undoes the effect of any prior
// calls to Notify for the provided signals.
// If no signals are provided, all incoming signals will be ignored.
func Ignore(sig ...os.Signal) {
	cancel(sig, ignoreSignal)
}

// Ignored reports whether sig is currently ignored.
func Ignored(sig os.Signal) bool {
	sn := signum(sig)
	return sn >= 0 && signalIgnored(sn)
}

var (
	// watchSignalLoopOnce guards calling the conditionally
	// initialized watchSignalLoop. If watchSignalLoop is non-nil,
	// it will be run in a goroutine lazily once Notify is invoked.
	// See Issue 21576.
	watchSignalLoopOnce sync.Once
	watchSignalLoop     func()
)

// Notify causes package signal to relay incoming signals to c.
// If no signals are provided, all incoming signals will be relayed to c.
// Otherwise, just the provided signals will.
//
// Package signal will not block sending to c: the caller must ensure
// that c has sufficient buffer space to keep up with the expected
// signal rate. For a channel used for notification of just one signal value,
// a buffer of size 1 is sufficient.
//
// It is allowed to call Notify multiple times with the same channel:
// each call expands the set of signals sent to that channel.
// The only way to remove signals from the set is to call Stop.
//
// It is allowed to call Notify multiple times with different channels
// and the same signals: each channel receives copies of incoming
// signals independently.
func Notify(c chan<- os.Signal, sig ...os.Signal) {
	if c == nil {
		panic("os/signal: Notify using nil channel")
	}

	handlers.Lock()
	defer handlers.Unlock()

	h := handlers.m[c]
	if h == nil {
		if handlers.m == nil {
			handlers.m = make(map[chan<- os.Signal]*handler)
		}
		h = new(handler)
		handlers.m[c] = h
	}

	add := func(n int) {
		// 信号无效
		if n < 0 {
			return
		}

		// 是否有n;
		if !h.want(n) {
			// 如果没有
			// 设置n
			h.set(n)

			// 如果信号的引用数是0
			if handlers.ref[n] == 0 {

				/// 激活信号
				enableSignal(n)

				// The runtime requires that we enable a
				// signal before starting the watcher.
				/// 启动观察器
				watchSignalLoopOnce.Do(func() {
					if watchSignalLoop != nil {
						go watchSignalLoop()
					}
				})
			}

			/// 增加引用数
			handlers.ref[n]++
		}
	}

	if len(sig) == 0 {
		for n := 0; n < numSig; n++ {
			add(n)
		}
	} else {
		for _, s := range sig {
			add(signum(s))
		}
	}
}

// Reset undoes the effect of any prior calls to Notify for the provided
// signals.
// If no signals are provided, all signal handlers will be reset.
func Reset(sig ...os.Signal) {
	cancel(sig, disableSignal)
}

// Stop causes package signal to stop relaying incoming signals to c.
// It undoes the effect of all prior calls to Notify using c.
// When Stop returns, it is guaranteed that c will receive no more signals.
func Stop(c chan<- os.Signal) {
	handlers.Lock()

	h := handlers.m[c]
	if h == nil {
		handlers.Unlock()
		return
	}
	delete(handlers.m, c)

	for n := 0; n < numSig; n++ {
		// 存在n
		if h.want(n) {
			handlers.ref[n]--

			// n == 0; 取消信号
			if handlers.ref[n] == 0 {
				disableSignal(n)
			}
		}
	}

	// Signals will no longer be delivered to the channel.
	// We want to avoid a race for a signal such as SIGINT:
	// it should be either delivered to the channel,
	// or the program should take the default action (that is, exit).
	// To avoid the possibility that the signal is delivered,
	// and the signal handler invoked, and then Stop deregisters
	// the channel before the process function below has a chance
	// to send it on the channel, put the channel on a list of
	// channels being stopped and wait for signal delivery to
	// quiesce before fully removing it.

	handlers.stopping = append(handlers.stopping, stopping{c, h})

	handlers.Unlock()

	/// 等待直到没有更多的信号被交付
	signalWaitUntilIdle()

	handlers.Lock()

	for i, s := range handlers.stopping {
		if s.c == c {
			handlers.stopping = append(handlers.stopping[:i], handlers.stopping[i+1:]...)
			break
		}
	}

	handlers.Unlock()
}

// Wait until there are no more signals waiting to be delivered.
// Defined by the runtime package.
func signalWaitUntilIdle()

func process(sig os.Signal) {
	n := signum(sig)
	if n < 0 {
		return
	}

	handlers.Lock()
	defer handlers.Unlock()

	for c, h := range handlers.m {
		/// 想要这个信号; 则发送
		if h.want(n) {
			// send but do not block for it
			select {
			case c <- sig:
			default:
			}
		}
	}

	/// 没有找到; 则说明再停止中，投递过去

	// Avoid the race mentioned in Stop.
	for _, d := range handlers.stopping {
		if d.h.want(n) {
			select {
			case d.c <- sig: /// 将信号发送到这个管道里
			default:
			}
		}
	}
}
