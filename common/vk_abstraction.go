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
	buffers, err := VKSAllocateCommandBuffers(device, &cbAllocateInfo)
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
	buffers, err := VKSAllocateCommandBuffers(device, &cbAllocateInfo)
	if err != nil {
		return nil, err
	}
	return buffers, nil
}

// VKBeginSingleTimeCommands allocates and starts the recording of a command buffer. This is meant to be followed closely
// by a call to VKEndSingleTimeCommands. It is intended to be used for a few ad-hoc commands that can be done quickly
// and don't need any a more proper handling of allocation and scheduling.
func VKBeginSingleTimeCommands(device vk.Device, pool vk.CommandPool) (vk.CommandBuffer, error) {
	cmdBuffers, err := VKAllocateCommandBuffersPrimary(device, pool, 1)
	if err != nil {
		return nil, err
	}
	beginInfo := vk.CommandBufferBeginInfo{
		SType:            vk.StructureTypeCommandBufferBeginInfo,
		PNext:            nil,
		Flags:            vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit),
		PInheritanceInfo: nil,
	}
	err = vk.Error(vk.BeginCommandBuffer(cmdBuffers[0], &beginInfo))
	if err != nil {
		return nil, err
	}
	return cmdBuffers[0], nil
}

// VKEndSingleTimeCommands is the second half started by VKBeginSingleTimeCommands. It should be called after execution
// of a single use command buffer. To do so, a few value are required: The device that it is on, the pool it was created
// from (needs to be the one provided as an argument to VKBeginSingleTimeCommands), the queue it will be submitted to and
// the buffer that is to be submitted / ended.
func VKEndSingleTimeCommands(device vk.Device, pool vk.CommandPool, queue vk.Queue, cmdBuffer vk.CommandBuffer) error {
	err := vk.Error(vk.EndCommandBuffer(cmdBuffer))
	if err != nil {
		return err
	}
	submitInfo := vk.SubmitInfo{
		SType:                vk.StructureTypeSubmitInfo,
		PNext:                nil,
		WaitSemaphoreCount:   0,
		PWaitSemaphores:      nil,
		PWaitDstStageMask:    nil,
		CommandBufferCount:   1,
		PCommandBuffers:      []vk.CommandBuffer{cmdBuffer},
		SignalSemaphoreCount: 0,
		PSignalSemaphores:    nil,
	}
	err = vk.Error(vk.QueueSubmit(queue, 1, []vk.SubmitInfo{submitInfo}, nil))
	if err != nil {
		return err
	}
	err = vk.Error(vk.QueueWaitIdle(queue))
	if err != nil {
		return err
	}
	vk.FreeCommandBuffers(device, pool, 1, []vk.CommandBuffer{cmdBuffer})
	return nil
}

func VKCreate2DFullSizeImageView(device vk.Device, image vk.Image, format vk.Format, aspectFlags vk.ImageAspectFlags) (vk.ImageView, error) {
	createInfo := &vk.ImageViewCreateInfo{
		SType:    vk.StructureTypeImageViewCreateInfo,
		PNext:    nil,
		Flags:    0,
		Image:    image,
		ViewType: vk.ImageViewType2d,
		Format:   format,
		Components: vk.ComponentMapping{
			R: vk.ComponentSwizzleIdentity,
			G: vk.ComponentSwizzleIdentity,
			B: vk.ComponentSwizzleIdentity,
			A: vk.ComponentSwizzleIdentity,
		},
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     aspectFlags,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	return VkCreateImageView(device, createInfo, nil)
}
