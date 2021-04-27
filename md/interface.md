# 接口

在`Go`语言中，接口分成两类：
1. 带方法的接口`iface`
2. 不带方法的接口`eface`

## 数据结构

~~~go
// 带方法的接口
type iface struct {
	tab  *itab          
	data unsafe.Pointer 
}
// 空接口
type eface struct {
	_type *_type         // 执向类型
	data  unsafe.Pointer // 指向底层数据
}
~~~

1. 空接口比较简单，只有一个指向类型的指针和一个执行底层数据的指针
2. 带方法的接口则包含一个指向`itab`的指针和指向底层数据的指针

~~~go
type _type struct {
	size       uintptr
	ptrdata    uintptr // 之前正在持有的指针，所占内存大小
	hash       uint32  // 进行快速的类型比较
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal func(unsafe.Pointer, unsafe.Pointer) bool
	// gcdata stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, gcdata is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	gcdata    *byte
	str       nameOff
	ptrToThis typeOff
}
~~~

以上是类型的结构体，这些字段有其特殊的用处。我们这里无需详细了解。

~~~go
type itab struct {
	inter *interfacetype
	_type *_type
	hash  uint32 // copy of _type.hash. Used for type switches.
	_     [4]byte
	fun   [1]uintptr // variable sized. fun[0]==0 means _type does not implement inter.
}
~~~

`itab`即表明了接口类型，也表明了具体的数据类型。

* `hash` : 类型比较
* `fun` : 是一个动态大小的数组，它是一个用于动态派发的虚函数表


