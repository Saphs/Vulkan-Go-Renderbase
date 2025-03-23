package model

import (
	"GPU_fluid_simulation/common"
	vk "github.com/goki/vulkan"
	"local/vector_math"
)

// UniformBufferObject a uniform buffer object as a tightly packed struct that will be transferred to the GPU.
type UniformBufferObject struct {
	View       vector_math.Mat
	Projection vector_math.Mat // 192byte calculated size
}

// SizeOfUbo returns size of the UniformBufferObject struct under the assumption that view and projection are
// 4x4 matrices. This is done due to the type system otherwise requiring us to implement all NxM matrices using
// fixed arrays instead of slices.
func SizeOfUbo() vk.DeviceSize {
	m, _ := vector_math.NewMat(4, 4)
	return vk.DeviceSize(m.ByteSize() * 2)
}

func (u *UniformBufferObject) Bytes() []byte {
	return append(common.RawBytes(u.View.Unroll()), common.RawBytes(u.Projection.Unroll())...)
}
