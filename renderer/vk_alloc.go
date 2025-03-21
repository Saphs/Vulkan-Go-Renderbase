package renderer

import (
	"GPU_fluid_simulation/common"
	vk "github.com/goki/vulkan"
	"log"
	"neilpa.me/go-stbi"
	"unsafe"
)

// This Code section contains allocation helper functions. It aims to simplify the allocation of buffers and
// images on the selected device.

type Buffer struct {
	handle    vk.Buffer
	deviceMem vk.DeviceMemory
	size      vk.DeviceSize
	usage     vk.BufferUsageFlags
	props     vk.MemoryPropertyFlags
}

func CreateBuffer(dc *common.DeviceContext, size vk.DeviceSize, usage vk.BufferUsageFlags, props vk.MemoryPropertyFlags) *Buffer {
	// Buffer handle of fitting size
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
	var buf vk.Buffer
	err := vk.Error(vk.CreateBuffer(dc.Device, &bufferInfo, nil, &buf))
	if err != nil {
		log.Panicf("Failed to create vertex buffer")
	}

	bufRequirements := readBufferMemoryRequirements(dc.Device, buf)

	// Allocate device memory
	allocInfo := vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		PNext:           nil,
		AllocationSize:  bufRequirements.Size,
		MemoryTypeIndex: findMemoryType(dc, bufRequirements.MemoryTypeBits, props),
	}
	var deviceMem vk.DeviceMemory
	err = vk.Error(vk.AllocateMemory(dc.Device, &allocInfo, nil, &deviceMem))
	if err != nil {
		log.Panicf("Failed to allocate vertex buffer memory")
	}

	// Associate allocated memory with buffer handle
	err = vk.Error(vk.BindBufferMemory(dc.Device, buf, deviceMem, 0))
	if err != nil {
		log.Panicf("Failed to bind device memory to buffer handle")
	}

	return &Buffer{
		handle:    buf,
		deviceMem: deviceMem,
		size:      size,
		usage:     usage,
		props:     props,
	}
}

// CopyToDeviceBuffer is a convenience method to simplify the process of mapping device memory to CPU memory,
// copy bytes over to the GPU and unmapping the memory again. This requires the buffer to:
// - have the stated usage: vk.BufferUsageTransferSrcBit
// - be: vk.MemoryPropertyHostVisibleBit and vk.MemoryPropertyHostCoherentBit
func CopyToDeviceBuffer(dc *common.DeviceContext, deviceBuf *Buffer, payload []byte) {
	// Check the memory is accessible by the CPU
	hasTransferUsage := deviceBuf.usage&vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit) != 0
	isHostVisCoh := deviceBuf.props&vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit) != 0
	if !(hasTransferUsage && isHostVisCoh) {
		log.Panicf("Cant copy to device buffer as buffer is not suitable")
	}
	// check for size mismatches - this function only allows to copy a "full buffer" worth of payload starting at offset = 0
	if deviceBuf.size != vk.DeviceSize(uint64(len(payload))) {
		log.Panicf("Cant copy to device buffer. Buffer and payload not of equal size.")
	}
	// Map -> copy -> Unmap
	var pData unsafe.Pointer
	err := vk.Error(vk.MapMemory(dc.Device, deviceBuf.deviceMem, 0, deviceBuf.size, 0, &pData))
	if err != nil {
		log.Panicf("Failed to map device memory")
	}
	vk.Memcopy(pData, payload)
	vk.UnmapMemory(dc.Device, deviceBuf.deviceMem)
}

func DestroyBuffer(dc *common.DeviceContext, buffer *Buffer) {
	vk.DestroyBuffer(dc.Device, buffer.handle, nil)
	vk.FreeMemory(dc.Device, buffer.deviceMem, nil)
}

type TextureImage struct {
	handle    vk.Image
	deviceMem vk.DeviceMemory
}

func CreateTextureImage(dc *common.DeviceContext, path string) *TextureImage {
	img, err := stbi.Load(path)
	if err != nil {
		log.Panicf("Failed to load %s: %v", path, err)
	}
	w := img.Rect.Dx()
	h := img.Rect.Dy()
	bytesPerPixel := 4
	imgSize := vk.DeviceSize(w * h * bytesPerPixel)
	log.Printf("Loaded image %s (w: %dp, h:%d) %d Byte", path, w, h, imgSize)

	stgBuf := CreateBuffer(
		dc,
		imgSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
	)

	// Map staging memory - copy our vertex data into staging - unmap staging again
	var pData unsafe.Pointer
	err = vk.Error(vk.MapMemory(dc.Device, stgBuf.deviceMem, 0, imgSize, 0, &pData))
	if err != nil {
		log.Panicf("Failed to map device memory")
	}
	vk.Memcopy(pData, img.Pix)
	vk.UnmapMemory(dc.Device, stgBuf.deviceMem)

	imgHandle, imgMem := CreateImage(
		dc,
		uint32(w),
		uint32(h),
		vk.FormatR8g8b8a8Srgb,
		vk.ImageTilingOptimal,
		vk.ImageUsageFlags(vk.ImageUsageTransferDstBit|vk.ImageUsageSampledBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit),
	)
	return &TextureImage{
		handle:    imgHandle,
		deviceMem: imgMem,
	}
}

func CreateImage(dc *common.DeviceContext, w uint32, h uint32, format vk.Format, tiling vk.ImageTiling, usage vk.ImageUsageFlags, props vk.MemoryPropertyFlags) (vk.Image, vk.DeviceMemory) {
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
	var img vk.Image
	if vk.CreateImage(dc.Device, imageInfo, nil, &img) != vk.Success {
		log.Panicf("failed to create image!")
	}
	memRequirements := readImageMemoryRequirements(dc.Device, img)
	allocInfo := &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		PNext:           nil,
		AllocationSize:  memRequirements.Size,
		MemoryTypeIndex: findMemoryType(dc, memRequirements.MemoryTypeBits, props),
	}
	var imgMemory vk.DeviceMemory
	if vk.AllocateMemory(dc.Device, allocInfo, nil, &imgMemory) != vk.Success {
		log.Panicf("failed to allocate device memory for image!")
	}
	vk.BindImageMemory(dc.Device, img, imgMemory, 0)
	return img, imgMemory
}

func findMemoryType(dc *common.DeviceContext, typeFilter uint32, propFlags vk.MemoryPropertyFlags) uint32 {
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
