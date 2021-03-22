// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package trace contains facilities for programs to generate traces
// for the Go execution tracer.
// trace 为Go执行跟踪器生成程序traces,
//
// Tracing runtime activities /// 跟踪运行时活动
//
// The execution trace captures a wide range of execution events such as
// goroutine creation/blocking/unblocking, syscall enter/exit/block,
// GC-related events, changes of heap size, processor start/stop, etc.
// A precise nanosecond-precision timestamp and a stack trace is
// captured for most events. The generated trace can be interpreted
// using `go tool trace`. /// 使用 go tool trace 进行交互
//
// Support for tracing tests and benchmarks built with the standard
// testing package is built into `go test`. For example, the following
// command runs the test in the current directory and writes the trace
// file (trace.out).
//
//    go test -trace=trace.out
/// 支持单元测试和性能测试；通过内建的标准测试包
/// 下列命令允许测试在当前目录，并我写入到trace文件trace.out
//
// This runtime/trace package provides APIs to add equivalent tracing
// support to a standalone program. See the Example that demonstrates
// how to use this API to enable tracing.       /// 足够的trace支持
//
// There is also a standard HTTP interface to trace data. Adding the
// following line will install a handler under the /debug/pprof/trace URL
// to download a live trace:
//
//     import _ "net/http/pprof"
//
// See the net/http/pprof package for more details about all of the
// debug endpoints installed by this import.
//
// User annotation /// 用户注释
//
// Package trace provides user annotation APIs that can be used to
// log interesting events during execution.      /// 打印事件日志
//
// There are three types of user annotations: log messages, regions,
// and tasks. /// 3种类型：日志消息，区域，任务
//
// Log emits a timestamped message to the execution trace along with
// additional information such as the category of the message and
// which goroutine called Log. The execution tracer provides UIs to filter
// and group goroutines using the log category and the message supplied
// in Log.
//
// A region is for logging a time interval during a goroutine's execution.
// By definition, a region starts and ends in the same goroutine. /// 启动和结束在相同的goroutine
// Regions can be nested to represent subintervals.   /// 区域可以被嵌套到维护格子内部
// For example, the following code records four regions in the execution
// trace to trace the durations of sequential steps in a cappuccino making
// operation.
/// 例如：下边的代码记录4个区域在执行程序的trace,去跟踪序列化步骤的持续时间在一个卡布奇诺[cappuccino]标记操作中
//
//   trace.WithRegion(ctx, "makeCappuccino", func() {
//
//      // orderID allows to identify a specific order
//      // among many cappuccino order region records.
//      trace.Log(ctx, "orderID", orderID)
//
//      trace.WithRegion(ctx, "steamMilk", steamMilk)
//      trace.WithRegion(ctx, "extractCoffee", extractCoffee)
//      trace.WithRegion(ctx, "mixMilkCoffee", mixMilkCoffee)
//   })
//
// A task is a higher-level component that aids tracing of logical
// operations such as an RPC request, an HTTP request, or an
// interesting local operation which may require multiple goroutines
// working together. Since tasks can involve multiple goroutines,
// they are tracked via a context.Context object. NewTask creates
// a new task and embeds it in the returned context.Context object.
// Log messages and regions are attached to the task, if any, in the
// Context passed to Log and WithRegion.
//
/// task 是一个高水平的组件，为了跟踪逻辑操作，例如：RPC请求，HTTP请求，或者一个内部的本地
/// 操作，其需要许多goroutine一起工作。task能够调用多个goroutine, 他们被跟踪通过context.Context对象。
/// NewTask创建一个新的Task; 内嵌于返回的context.Context对象中；Log Message 和 regions 依附在task上
/// Context 被传递到Log和WithRegion

// For example, assume that we decided to froth milk, extract coffee,
// and mix milk and coffee in separate goroutines. With a task,
// the trace tool can identify the goroutines involved in a specific
// cappuccino order.
//
//      ctx, task := trace.NewTask(ctx, "makeCappuccino")
//      trace.Log(ctx, "orderID", orderID)
//
//      milk := make(chan bool)
//      espresso := make(chan bool)
//
//      go func() {
//              trace.WithRegion(ctx, "steamMilk", steamMilk)
//              milk <- true
//      }()
//      go func() {
//              trace.WithRegion(ctx, "extractCoffee", extractCoffee)
//              espresso <- true
//      }()
//      go func() {
//              defer task.End() // When assemble is done, the order is complete.
//              <-espresso
//              <-milk
//              trace.WithRegion(ctx, "mixMilkCoffee", mixMilkCoffee)
//      }()
//
//
// The trace tool computes the latency of a task by measuring the
// time between the task creation and the task end and provides
// latency distributions for each task type found in the trace.
package trace

import (
	"io"
	"runtime"
	"sync"
	"sync/atomic"
)

// Start enables tracing for the current program.
// While tracing, the trace will be buffered and written to w.
// Start returns an error if tracing is already enabled.
func Start(w io.Writer) error {
	tracing.Lock()
	defer tracing.Unlock()

	if err := runtime.StartTrace(); err != nil {
		return err
	}
	go func() {
		for {
			data := runtime.ReadTrace()
			if data == nil {
				break
			}
			w.Write(data)
		}
	}()
	atomic.StoreInt32(&tracing.enabled, 1)
	return nil
}

// Stop stops the current tracing, if any.
// Stop only returns after all the writes for the trace have completed.
func Stop() {
	tracing.Lock()
	defer tracing.Unlock()
	atomic.StoreInt32(&tracing.enabled, 0)

	runtime.StopTrace()
}

var tracing struct {
	sync.Mutex       // gate mutators (Start, Stop)
	enabled    int32 // accessed via atomic
}
