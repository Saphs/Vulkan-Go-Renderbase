package common

import (
	vk "github.com/goki/vulkan"
)

// Utility functions providing slightly altered versions of the raw go bindings and wrapped functions. These altered
// versions of common functions should only hide very obvious default values that will not need to change most of the
// time. Thus representing a tiny step-up in abstraction to allow for a simpler usage of common vulkan calls. Each
// simplification function should specify the simplification it does. Names are prefixed with VKS which stands for
// (V)ul(K)an (S)implified.

// VKSAllocateCommandBuffers simplifies vk.AllocateCommandBuffers(...) by assuming the number of desired CommandBuffers
// to create is provided in the vk.CommandBufferAllocateInfo parameter.
func VKSAllocateCommandBuffers(device vk.Device, pAllocateInfo *vk.CommandBufferAllocateInfo) ([]vk.CommandBuffer, error) {
	var buffers = make([]vk.CommandBuffer, pAllocateInfo.CommandBufferCount)
	err := vk.Error(vk.AllocateCommandBuffers(device, pAllocateInfo, buffers))
	if err != nil {
		return nil, err
	}
	return buffers, nil
}

// VKSCreateCommandPool implicitly instantiates the CreateInfo for the command pool based in the provided arguments. This
// is easily possible as the CreateInfo does only contain 2 interesting value sin this case.
func VKSCreateCommandPool(device vk.Device, flags vk.CommandPoolCreateFlags, QueueFamilyIndex uint32) (vk.CommandPool, error) {
	poolInfo := vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		PNext:            nil,
		Flags:            flags,
		QueueFamilyIndex: QueueFamilyIndex,
	}
	return VkCreateCommandPool(device, &poolInfo, nil)
}
