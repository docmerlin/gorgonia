package gorgonia

import "unsafe"

// Memory is a representation of memory of the value.
//
// The main reason for requiring both Uintptr() and Pointer() methods is because while Go currently does not have a compacting
// garbage collector, from the docs of `unsafe`:
//		Even if a uintptr holds the address of some object, the garbage collector, will not update that uintptr's value if the object moves,
//		nor will that uintptr keep the object from being reclaimed.
type Memory interface {
	Uintptr() uintptr
	MemSize() uintptr
	Pointer() unsafe.Pointer
}

// Arena is a representation of a pool of Memory
type Arena interface {
	Get(dev Device, size int64) (Memory, error)                 // Get returns a NoOpError when it cannot get a memory. Please allocate
	GetAndCopy(dev Device, size int64, v Value) (Memory, error) // Gets a memory and copies the values into the memory and returns it.
	Put(dev Device, mem Memory, size int64)                     // puts the memory back into the arena
}

// External is a representation of an external device (cuda/cgo/openCL), conceptually modelled as a machine.
type External interface {
	Arena
	HasFunc(string) bool
	Signal() // signals the machine to do work
	Sync() chan struct{}
}

// ExecutionContext informs how an op should be executed
type ExecutionContext struct {
	External
	Device
}

// ExternalOp is an op that contains an external context. This allows for ops to be run without needing a VM
type ExternalOp struct {
	Op
	ExecutionContext

	Prealloc  Value
	UseUnsafe bool
}

// NewExternalOp creates a new *ExternalOp.
func NewExternalOp(op Op, ctx ExecutionContext, prealloc Value) *ExternalOp {
	return &ExternalOp{
		Op:               op,
		ExecutionContext: ctx,
		Prealloc:         prealloc,
		UseUnsafe:        false,
	}
}

// Do performs the op,
func (op *ExternalOp) Do(vals ...Value) (Value, error) {
	switch o := op.Op.(type) {
	case CUDADoer:
		if op.UseUnsafe || op.Prealloc == nil {
			return o.CUDADo(op.External, op.Device, vals[0], vals...)
		}
		return o.CUDADo(op.External, op.Device, op.Prealloc, vals...)
	case CLDoer:
	case UnsafeDoer:
		if op.UseUnsafe {
			return o.UnsafeDo(vals...)
		}
		return op.Do(vals...)
	case UsePreallocDoer:
		return o.UsePreallocDo(op.Prealloc, vals...)
	default:
		return o.Do(vals...)
	}
	panic("Unreachable")
}

func (op *ExternalOp) String() string {
	return op.Op.String()
}
