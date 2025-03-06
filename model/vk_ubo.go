package model

import (
	"GPU_fluid_simulation/tooling"
	"local/vector_math"
)

type UniformBufferObject struct {
	View       vector_math.Mat
	Projection vector_math.Mat // 192byte calculated size
}

// SizeOfUbo returns size of the UniformBufferObject struct under the assumption
// that model, view and projection are 4x4 matrices. This is done due to some
// the type system otherwise requiring us to implement all NxM matrices using
// fixed arrays instead of slices.
func SizeOfUbo() uintptr {
	m, _ := vector_math.NewMat(4, 4)
	return uintptr(m.ByteSize() * 2)
}

func (u *UniformBufferObject) Bytes() []byte {
	return append(append(tooling.ToByteArr(u.View.Unroll())), tooling.ToByteArr(u.Projection.Unroll())...)
}
