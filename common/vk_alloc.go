package common

import (
	vk "github.com/goki/vulkan"
	"log"
)

// This Code section contains allocation helper functions. It aims to simplify the allocation of buffers and
// images on the selected device.

type Buffer struct {
	Handle    vk.Buffer
	DeviceMem vk.DeviceMemory
	Size      vk.DeviceSize
	Usage     vk.BufferUsageFlags
	props     vk.MemoryPropertyFlags
}

func CreateBuffer(dc *Device, size vk.DeviceSize, usage vk.BufferUsageFlags, props vk.MemoryPropertyFlags) *Buffer {
	// Buffer Handle of fitting Size
	bufferInfo := vk.BufferCreateInfo{
		SType:                 vk.StructureTypeBufferCreateInfo,
		PNext:                 nil,
		Flags:                 0,
		Size:                  size,
		Usage:                 usage,
		SharingMode:           vk.SharingModeExclusive,
		QueueFamilyIndexCount: 0,
		PQueueFamilyIndices:   nil,
	}

	buf, err := VkCreateBuffer(dc.D, &bufferInfo, nil)
	if err != nil {
		log.Panicf("Failed to create vertex buffer")
	}

	bufRequirements := ReadBufferMemoryRequirements(dc.D, buf)

	// Allocate device memory
	allocInfo := vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		PNext:           nil,
		AllocationSize:  bufRequirements.Size,
		MemoryTypeIndex: findMemoryType(dc, bufRequirements.MemoryTypeBits, props),
	}
	deviceMem, err := VkAllocateMemory(dc.D, &allocInfo, nil)
	if err != nil {
		log.Panicf("Failed to allocate vertex buffer memory")
	}

	// Associate allocated memory with buffer Handle
	err = VkBindBufferMemory(dc.D, buf, deviceMem, 0)
	if err != nil {
		log.Panicf("Failed to bind device memory to buffer Handle")
	}

	return &Buffer{
		Handle:    buf,
		DeviceMem: deviceMem,
		Size:      size,
		Usage:     usage,
		props:     props,
	}
}

// CopyToDeviceBuffer is a convenience method to simplify the process of mapping device memory to CPU memory,
// copy bytes over to the GPU and unmapping the memory again. This requires the buffer to:
// - have the stated Usage: vk.BufferUsageTransferSrcBit
// - be: vk.MemoryPropertyHostVisibleBit and vk.MemoryPropertyHostCoherentBit
func CopyToDeviceBuffer(dc *Device, deviceBuf *Buffer, payload []byte) {
	// Check the memory is accessible by the CPU
	hasTransferUsage := deviceBuf.Usage&vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit) != 0
	isHostVisCoh := deviceBuf.props&vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit) != 0
	if !(hasTransferUsage && isHostVisCoh) {
		log.Panicf("Cant copy to device buffer as buffer is not suitable")
	}
	// check for Size mismatches - this function only allows to copy a "full buffer" worth of payload starting at offset = 0
	if deviceBuf.Size != vk.DeviceSize(uint64(len(payload))) {
		log.Panicf("Cant copy to device buffer. Buffer and payload not of equal Size.")
	}
	// Map -> copy -> Unmap
	pData, err := VkMapMemory(dc.D, deviceBuf.DeviceMem, 0, deviceBuf.Size, 0)
	if err != nil {
		log.Panicf("Failed to map device memory")
	}
	bCopied := vk.Memcopy(pData, payload)
	log.Printf("copied %d bytes from cpu to device", bCopied)
	vk.UnmapMemory(dc.D, deviceBuf.DeviceMem)
}

func DestroyBuffer(dc *Device, buffer *Buffer) {
	vk.DestroyBuffer(dc.D, buffer.Handle, nil)
	vk.FreeMemory(dc.D, buffer.DeviceMem, nil)
}

type TextureImage struct {
	handle    vk.Image
	deviceMem vk.DeviceMemory
}

func CreateImage(dc *Device, w uint32, h uint32, format vk.Format, tiling vk.ImageTiling, usage vk.ImageUsageFlags, props vk.MemoryPropertyFlags) (vk.Image, vk.DeviceMemory) {
	imageInfo := &vk.ImageCreateInfo{
		SType:     vk.StructureTypeImageCreateInfo,
		PNext:     nil,
		Flags:     0,
		ImageType: vk.ImageType2d,
		Format:    format,
		Extent: vk.Extent3D{
			Width:  w,
			Height: h,
			Depth:  1,
		},
		MipLevels:             1,
		ArrayLayers:           1,
		Samples:               vk.SampleCount1Bit,
		Tiling:                tiling,
		Usage:                 usage,
		SharingMode:           vk.SharingModeExclusive,
		QueueFamilyIndexCount: 0,
		PQueueFamilyIndices:   nil,
		InitialLayout:         vk.ImageLayoutUndefined,
	}
	img, err := VkCreateImage(dc.D, imageInfo, nil)
	if err != nil {
		log.Panicf("failed to create image!")
	}

	memRequirements := ReadImageMemoryRequirements(dc.D, img)
	allocInfo := &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		PNext:           nil,
		AllocationSize:  memRequirements.Size,
		MemoryTypeIndex: findMemoryType(dc, memRequirements.MemoryTypeBits, props),
	}
	imgMemory, err := VkAllocateMemory(dc.D, allocInfo, nil)
	if err != nil {
		log.Panicf("Failed to allocate image device memory")
	}
	vk.BindImageMemory(dc.D, img, imgMemory, 0)
	return img, imgMemory
}

func findMemoryType(dc *Device, typeFilter uint32, propFlags vk.MemoryPropertyFlags) uint32 {
	//log.Printf("Got memory properties: %v", toStringPhysicalDeviceMemProps(c.pdMemoryProps))
	for i := uint32(0); i < dc.PdMemoryProps.MemoryTypeCount; i++ {
		ofType := (typeFilter & (1 << i)) > 0
		hasProperties := dc.PdMemoryProps.MemoryTypes[i].PropertyFlags&propFlags == propFlags
		if ofType && hasProperties {
			log.Printf("Found memory type for buffer -> %d on heap %d", i, dc.PdMemoryProps.MemoryTypes[i].HeapIndex)
			return i
		}
	}
	log.Panicf("Failed to find suitable memory type")
	return 0
}
