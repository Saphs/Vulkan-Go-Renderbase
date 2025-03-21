package renderer

import (
	vk "github.com/goki/vulkan"
)

// Read operations that require duplicated function calls, allocations and dereferencing. It is pulled out to
// provide a more go-lang feel and tidy the core code.

func readBufferMemoryRequirements(device vk.Device, b vk.Buffer) vk.MemoryRequirements {
	var memRequirements vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(device, b, &memRequirements)
	memRequirements.Deref()
	return memRequirements
}

func readImageMemoryRequirements(device vk.Device, img vk.Image) vk.MemoryRequirements {
	var memRequirements vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, img, &memRequirements)
	memRequirements.Deref()
	return memRequirements
}
