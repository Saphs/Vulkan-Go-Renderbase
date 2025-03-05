package model

import (
	"local/vector_math"
	"unsafe"
)

type UniformBufferObject struct {
	Model      vector_math.Mat // 64byte (8 time 8byte words, no padding)
	View       vector_math.Mat
	Projection vector_math.Mat // 192byte calculated size
}

// SizeOfUbo returns size of the UniformBufferObject struct under the assumption
// that model, view and projection are 4x4 matrices. This is done due to some
// the type system otherwise requiring us to implement all NxM matrices using
// fixed arrays instead of slices.
func SizeOfUbo() uintptr {
	m, _ := vector_math.NewMat(4, 4)
	return uintptr(m.ByteSize() * 3)
}

func toByteArr(in []float32) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&in[0])), len(in)*4)
}

func (u *UniformBufferObject) Bytes() []byte {
	return append(append(toByteArr(u.Model.Unroll()), toByteArr(u.View.Unroll())...), toByteArr(u.Projection.Unroll())...)
}
