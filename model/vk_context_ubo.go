package model

import (
	"GPU_fluid_simulation/common"
	vk "github.com/goki/vulkan"
)

// ContextUniformBufferObject a uniform buffer object as a tightly packed struct that will be transferred to the GPU.
// This one contains context information for each model and will be bound for each model between draw calls.
type ContextUniformBufferObject struct {
	ModelType uint32
}

// SizeOfCtxUbo returns size of the ContextUniformBufferObject
func SizeOfCtxUbo() vk.DeviceSize {
	return vk.DeviceSize(4)
}

func (u *ContextUniformBufferObject) Bytes() []byte {
	return common.RawBytes(u)
}
