package common

import (
	vk "github.com/goki/vulkan"
)

// Utility functions that reduce visual clutter by abstracting some of the common default values into very obvious
// functions that should cover their respective use case most of the time. This is done to cut down on labor writing
// things out that are unlikely to change or are not relevant now. The main way typing is reduced by moving or
// defaulting parameters from 'createInfo' structs.

func VKAllocateCommandBuffersPrimary(device vk.Device, cmdPool vk.CommandPool, count uint32) ([]vk.CommandBuffer, error) {
	cbAllocateInfo := vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		PNext:              nil,
		CommandPool:        cmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: count,
	}
	buffers, err := VKAllocateCommandBuffers(device, &cbAllocateInfo)
	if err != nil {
		return nil, err
	}
	return buffers, nil
}

func VKAllocateCommandBuffersSecondary(device vk.Device, cmdPool vk.CommandPool, count uint32) ([]vk.CommandBuffer, error) {
	cbAllocateInfo := vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		PNext:              nil,
		CommandPool:        cmdPool,
		Level:              vk.CommandBufferLevelSecondary,
		CommandBufferCount: count,
	}
	buffers, err := VKAllocateCommandBuffers(device, &cbAllocateInfo)
	if err != nil {
		return nil, err
	}
	return buffers, nil
}
