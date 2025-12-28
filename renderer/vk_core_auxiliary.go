package renderer

import (
	com "GPU_fluid_simulation/common"
	"log"

	vk "github.com/goki/vulkan"
)

// These functions auxiliary functions that abstract from the raw Vulkan API by assuming some reasonable
// defaults where possible. These differ from the VKS function in vk_simplifications.go by being tied to a given
// Core Struct and are closer to helper function in the class than being a general abstraction of the API.

// allocDescriptorSets Allocates a list of descriptor sets of given layout from the stated pool
func (c *Core) allocDescriptorSets(pool vk.DescriptorPool, layouts []vk.DescriptorSetLayout) []vk.DescriptorSet {
	cnt := uint32(len(layouts))
	allocInfo := vk.DescriptorSetAllocateInfo{
		SType:              vk.StructureTypeDescriptorSetAllocateInfo,
		PNext:              nil,
		DescriptorPool:     pool,
		DescriptorSetCount: cnt,
		PSetLayouts:        layouts,
	}
	sets := make([]vk.DescriptorSet, cnt)
	err := vk.Error(vk.AllocateDescriptorSets(c.device.D, &allocInfo, &(sets[0])))
	if err != nil {
		log.Panicf("Failed to allocate descriptor sets")
	}
	return sets
}

func (c *Core) beginSingleTimeCommands() vk.CommandBuffer {
	cmdBuffer, err := com.VKBeginSingleTimeCommands(c.device.D, c.commandPool)
	if err != nil {
		log.Panicf("Failed to create command buffer for single time use: %v", err)
	}
	return cmdBuffer
}

func (c *Core) endSingleTimeCommands(cmdBuf vk.CommandBuffer, queue vk.Queue) {
	err := com.VKEndSingleTimeCommands(c.device.D, c.commandPool, queue, cmdBuf)
	if err != nil {
		log.Panicf("Failed to end single time use command buffer: %v", err)
	}
}

// copyVkBuffer is a subroutine that prepares a command buffer that is then executed on the device.
// The command buffer is allocated, records the copy command and is submitted to the device. After idle
// the command buffer is freed.
func (c *Core) copyVkBuffer(src vk.Buffer, dst vk.Buffer, s vk.DeviceSize) {
	cmdBuf := c.beginSingleTimeCommands()
	copyRegions := []vk.BufferCopy{
		{
			SrcOffset: 0,
			DstOffset: 0,
			Size:      s,
		},
	}
	vk.CmdCopyBuffer(cmdBuf, src, dst, 1, copyRegions)
	c.endSingleTimeCommands(cmdBuf, c.device.GraphicsQ)
}

func (c *Core) copyBuffer(src *com.Buffer, dst *com.Buffer, s vk.DeviceSize) {
	c.copyVkBuffer(src.Handle, dst.Handle, s)
}
