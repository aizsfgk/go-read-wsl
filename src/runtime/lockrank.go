// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file records the static ranks of the locks in the runtime. If a lock
// is not given a rank, then it is assumed to be a leaf lock, which means no other
// lock can be acquired while it is held. Therefore, leaf locks do not need to be
// given an explicit rank. We list all of the architecture-independent leaf locks
// for documentation purposes, but don't list any of the architecture-dependent
// locks (which are all leaf locks). debugLock is ignored for ranking, since it is used
// when printing out lock ranking errors.
/// 这个文件记录了运行时锁的静态排序，如果一个锁没有给一个rank, 它被认为是一个叶子锁，这意味着
/// 没有其他锁能被获取，当它被持有时。因此，叶子锁不需要给一个明确的排序。我们列出了所有的架构独立的叶子锁
/// 为了文档意图。但是不要列出架构依赖的锁。debugLock被忽视排序，尽管它被使用，当打印输出锁错误。
///
///
// lockInit(l *mutex, rank int) is used to set the rank of lock before it is used.
// If there is no clear place to initialize a lock, then the rank of a lock can be
// specified during the lock call itself via lockWithrank(l *mutex, rank int).
/// lockInit 被用来设置锁排序在使用之前
/// 如果这里不清晰的地方去初始化一个锁，排序锁能被定义，在锁被自身调用通过lockWithrank(l *mutex, rank int)
//
// Besides the static lock ranking (which is a total ordering of the locks), we
// also represent and enforce the actual partial order among the locks in the
// arcs[] array below. That is, if it is possible that lock B can be acquired when
// lock A is the previous acquired lock that is still held, then there should be
// an entry for A in arcs[B][]. We will currently fail not only if the total order
// (the lock ranking) is violated, but also if there is a missing entry in the
// partial order.

/// 总排序锁被违反则失败

package runtime

type lockRank int

// Constants representing the lock rank of the architecture-independent locks in
// the runtime. Locks with lower rank must be taken before locks with higher
// rank.
/// 等级交低的锁必须在等级较高的锁之前取下
const (
	lockRankDummy lockRank = iota

	// Locks held above sched
	lockRankSysmon
	lockRankScavenge
	lockRankForcegc
	lockRankSweepWaiters
	lockRankAssistQueue
	lockRankCpuprof
	lockRankSweep

	lockRankSched
	lockRankDeadlock
	lockRankPanic
	lockRankAllg
	lockRankAllp
	lockRankPollDesc

	lockRankTimers // Multiple timers locked simultaneously in destroy()
	lockRankItab
	lockRankReflectOffs
	lockRankHchan // Multiple hchans acquired in lock order in syncadjustsudogs()
	lockRankFin
	lockRankNotifyList
	lockRankTraceBuf
	lockRankTraceStrings
	lockRankMspanSpecial
	lockRankProf
	lockRankGcBitsArenas
	lockRankRoot
	lockRankTrace
	lockRankTraceStackTab
	lockRankNetpollInit

	lockRankRwmutexW
	lockRankRwmutexR

	lockRankMcentral // For !go115NewMCentralImpl
	lockRankSpine    // For !go115NewMCentralImpl
	lockRankSpanSetSpine
	lockRankGscan
	lockRankStackpool
	lockRankStackLarge
	lockRankDefer
	lockRankSudog

	// Memory-related non-leaf locks
	lockRankWbufSpans
	lockRankMheap
	lockRankMheapSpecial

	// Memory-related leaf locks
	lockRankGlobalAlloc

	// Other leaf locks
	lockRankGFree
	// Generally, hchan must be acquired before gscan. But in one specific
	// case (in syncadjustsudogs from markroot after the g has been suspended
	// by suspendG), we allow gscan to be acquired, and then an hchan lock. To
	// allow this case, we get this lockRankHchanLeaf rank in
	// syncadjustsudogs(), rather than lockRankHchan. By using this special
	// rank, we don't allow any further locks to be acquired other than more
	// hchan locks.
	lockRankHchanLeaf

	// Leaf locks with no dependencies, so these constants are not actually used anywhere.
	// There are other architecture-dependent leaf locks as well.
	lockRankNewmHandoff
	lockRankDebugPtrmask
	lockRankFaketimeState
	lockRankTicks
	lockRankRaceFini
	lockRankPollCache
	lockRankDebug
)

// lockRankLeafRank is the rank of lock that does not have a declared rank, and hence is
// a leaf lock.
const lockRankLeafRank lockRank = 1000

// lockNames gives the names associated with each of the above ranks
var lockNames = []string{
	lockRankDummy: "",

	lockRankSysmon:       "sysmon",
	lockRankScavenge:     "scavenge",
	lockRankForcegc:      "forcegc",
	lockRankSweepWaiters: "sweepWaiters",
	lockRankAssistQueue:  "assistQueue",
	lockRankCpuprof:      "cpuprof",
	lockRankSweep:        "sweep",

	lockRankSched:    "sched",
	lockRankDeadlock: "deadlock",
	lockRankPanic:    "panic",
	lockRankAllg:     "allg",
	lockRankAllp:     "allp",
	lockRankPollDesc: "pollDesc",

	lockRankTimers:      "timers",
	lockRankItab:        "itab",
	lockRankReflectOffs: "reflectOffs",

	lockRankHchan:         "hchan",
	lockRankFin:           "fin",
	lockRankNotifyList:    "notifyList",
	lockRankTraceBuf:      "traceBuf",
	lockRankTraceStrings:  "traceStrings",
	lockRankMspanSpecial:  "mspanSpecial",
	lockRankProf:          "prof",
	lockRankGcBitsArenas:  "gcBitsArenas",
	lockRankRoot:          "root",
	lockRankTrace:         "trace",
	lockRankTraceStackTab: "traceStackTab",
	lockRankNetpollInit:   "netpollInit",

	lockRankRwmutexW: "rwmutexW",
	lockRankRwmutexR: "rwmutexR",

	lockRankMcentral:     "mcentral",
	lockRankSpine:        "spine",
	lockRankSpanSetSpine: "spanSetSpine",
	lockRankGscan:        "gscan",
	lockRankStackpool:    "stackpool",
	lockRankStackLarge:   "stackLarge",
	lockRankDefer:        "defer",
	lockRankSudog:        "sudog",

	lockRankWbufSpans:    "wbufSpans",
	lockRankMheap:        "mheap",
	lockRankMheapSpecial: "mheapSpecial",

	lockRankGlobalAlloc: "globalAlloc.mutex",

	lockRankGFree:     "gFree",
	lockRankHchanLeaf: "hchanLeaf",

	lockRankNewmHandoff:   "newmHandoff.lock",
	lockRankDebugPtrmask:  "debugPtrmask.lock",
	lockRankFaketimeState: "faketimeState.lock",
	lockRankTicks:         "ticks.lock",
	lockRankRaceFini:      "raceFiniLock",
	lockRankPollCache:     "pollCache.lock",
	lockRankDebug:         "debugLock",
}

func (rank lockRank) String() string {
	if rank == 0 {
		return "UNKNOWN"
	}
	if rank == lockRankLeafRank {
		return "LEAF"
	}
	return lockNames[rank]
}

// lockPartialOrder is a partial order among the various lock types, listing the immediate
// ordering that has actually been observed in the runtime. Each entry (which
// corresponds to a particular lock rank) specifies the list of locks that can be
// already be held immediately "above" it.
//
// So, for example, the lockRankSched entry shows that all the locks preceding it in
// rank can actually be held. The fin lock shows that only the sched, timers, or
// hchan lock can be held immediately above it when it is acquired.
var lockPartialOrder [][]lockRank = [][]lockRank{
	lockRankDummy:         {},
	lockRankSysmon:        {},
	lockRankScavenge:      {lockRankSysmon},
	lockRankForcegc:       {lockRankSysmon},
	lockRankSweepWaiters:  {},
	lockRankAssistQueue:   {},
	lockRankCpuprof:       {},
	lockRankSweep:         {},
	lockRankSched:         {lockRankSysmon, lockRankScavenge, lockRankForcegc, lockRankSweepWaiters, lockRankAssistQueue, lockRankCpuprof, lockRankSweep},
	lockRankDeadlock:      {lockRankDeadlock},
	lockRankPanic:         {lockRankDeadlock},
	lockRankAllg:          {lockRankSysmon, lockRankSched, lockRankPanic},
	lockRankAllp:          {lockRankSysmon, lockRankSched},
	lockRankPollDesc:      {},
	lockRankTimers:        {lockRankSysmon, lockRankScavenge, lockRankSched, lockRankAllp, lockRankPollDesc, lockRankTimers},
	lockRankItab:          {},
	lockRankReflectOffs:   {lockRankItab},
	lockRankHchan:         {lockRankScavenge, lockRankSweep, lockRankHchan},
	lockRankFin:           {lockRankSysmon, lockRankScavenge, lockRankSched, lockRankAllg, lockRankTimers, lockRankHchan},
	lockRankNotifyList:    {},
	lockRankTraceBuf:      {lockRankSysmon, lockRankScavenge},
	lockRankTraceStrings:  {lockRankTraceBuf},
	lockRankMspanSpecial:  {lockRankSysmon, lockRankScavenge, lockRankAssistQueue, lockRankCpuprof, lockRankSched, lockRankAllg, lockRankAllp, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankHchan, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings},
	lockRankProf:          {lockRankSysmon, lockRankScavenge, lockRankAssistQueue, lockRankCpuprof, lockRankSweep, lockRankSched, lockRankAllg, lockRankAllp, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankHchan},
	lockRankGcBitsArenas:  {lockRankSysmon, lockRankScavenge, lockRankAssistQueue, lockRankCpuprof, lockRankSched, lockRankAllg, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankHchan},
	lockRankRoot:          {},
	lockRankTrace:         {lockRankSysmon, lockRankScavenge, lockRankForcegc, lockRankAssistQueue, lockRankSched, lockRankHchan, lockRankTraceBuf, lockRankTraceStrings, lockRankRoot, lockRankSweep},
	lockRankTraceStackTab: {lockRankScavenge, lockRankSweepWaiters, lockRankAssistQueue, lockRankSweep, lockRankSched, lockRankAllg, lockRankTimers, lockRankHchan, lockRankFin, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankRoot, lockRankTrace},
	lockRankNetpollInit:   {lockRankTimers},

	lockRankRwmutexW: {},
	lockRankRwmutexR: {lockRankRwmutexW},

	lockRankMcentral:     {lockRankSysmon, lockRankScavenge, lockRankForcegc, lockRankAssistQueue, lockRankCpuprof, lockRankSweep, lockRankSched, lockRankAllg, lockRankAllp, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankHchan},
	lockRankSpine:        {lockRankSysmon, lockRankScavenge, lockRankAssistQueue, lockRankCpuprof, lockRankSched, lockRankAllg, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankHchan},
	lockRankSpanSetSpine: {lockRankSysmon, lockRankScavenge, lockRankForcegc, lockRankAssistQueue, lockRankCpuprof, lockRankSweep, lockRankSched, lockRankAllg, lockRankAllp, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankHchan},
	lockRankGscan:        {lockRankSysmon, lockRankScavenge, lockRankForcegc, lockRankSweepWaiters, lockRankAssistQueue, lockRankCpuprof, lockRankSweep, lockRankSched, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankHchan, lockRankFin, lockRankTraceBuf, lockRankTraceStrings, lockRankRoot, lockRankNotifyList, lockRankProf, lockRankGcBitsArenas, lockRankTrace, lockRankTraceStackTab, lockRankNetpollInit, lockRankMcentral, lockRankSpine, lockRankSpanSetSpine},
	lockRankStackpool:    {lockRankSysmon, lockRankScavenge, lockRankSweepWaiters, lockRankAssistQueue, lockRankCpuprof, lockRankSweep, lockRankSched, lockRankPollDesc, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankHchan, lockRankFin, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankProf, lockRankGcBitsArenas, lockRankRoot, lockRankTrace, lockRankTraceStackTab, lockRankNetpollInit, lockRankRwmutexR, lockRankMcentral, lockRankSpine, lockRankSpanSetSpine, lockRankGscan},
	lockRankStackLarge:   {lockRankSysmon, lockRankAssistQueue, lockRankSched, lockRankItab, lockRankHchan, lockRankProf, lockRankGcBitsArenas, lockRankRoot, lockRankMcentral, lockRankSpanSetSpine, lockRankGscan},
	lockRankDefer:        {},
	lockRankSudog:        {lockRankNotifyList, lockRankHchan},
	lockRankWbufSpans:    {lockRankSysmon, lockRankScavenge, lockRankSweepWaiters, lockRankAssistQueue, lockRankSweep, lockRankSched, lockRankAllg, lockRankPollDesc, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankHchan, lockRankNotifyList, lockRankTraceStrings, lockRankMspanSpecial, lockRankProf, lockRankRoot, lockRankGscan, lockRankDefer, lockRankSudog},
	lockRankMheap:        {lockRankSysmon, lockRankScavenge, lockRankSweepWaiters, lockRankAssistQueue, lockRankCpuprof, lockRankSweep, lockRankSched, lockRankAllg, lockRankAllp, lockRankPollDesc, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankHchan, lockRankMspanSpecial, lockRankProf, lockRankGcBitsArenas, lockRankRoot, lockRankMcentral, lockRankGscan, lockRankStackpool, lockRankStackLarge, lockRankDefer, lockRankSudog, lockRankWbufSpans, lockRankSpanSetSpine},
	lockRankMheapSpecial: {lockRankSysmon, lockRankScavenge, lockRankAssistQueue, lockRankCpuprof, lockRankSweep, lockRankSched, lockRankAllg, lockRankAllp, lockRankTimers, lockRankItab, lockRankReflectOffs, lockRankNotifyList, lockRankTraceBuf, lockRankTraceStrings, lockRankHchan},
	lockRankGlobalAlloc:  {lockRankProf, lockRankSpine, lockRankSpanSetSpine, lockRankMheap, lockRankMheapSpecial},

	lockRankGFree:     {lockRankSched},
	lockRankHchanLeaf: {lockRankGscan, lockRankHchanLeaf},

	lockRankNewmHandoff:   {},
	lockRankDebugPtrmask:  {},
	lockRankFaketimeState: {},
	lockRankTicks:         {},
	lockRankRaceFini:      {},
	lockRankPollCache:     {},
	lockRankDebug:         {},
}
