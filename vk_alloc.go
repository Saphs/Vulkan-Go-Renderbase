package main

import (
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
}

func CreateBuffer(dc *DeviceContext, size vk.DeviceSize, usage vk.BufferUsageFlags, props vk.MemoryPropertyFlags) *Buffer {
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
	err := vk.Error(vk.CreateBuffer(dc.device, &bufferInfo, nil, &buf))
	if err != nil {
		log.Panicf("Failed to create vertex buffer")
	}

	bufRequirements := readBufferMemoryRequirements(dc.device, buf)

	// Allocate device memory
	allocInfo := vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		PNext:           nil,
		AllocationSize:  bufRequirements.Size,
		MemoryTypeIndex: findMemoryType(dc, bufRequirements.MemoryTypeBits, props),
	}
	var deviceMem vk.DeviceMemory
	err = vk.Error(vk.AllocateMemory(dc.device, &allocInfo, nil, &deviceMem))
	if err != nil {
		log.Panicf("Failed to allocate vertex buffer memory")
	}

	// Associate allocated memory with buffer handle
	err = vk.Error(vk.BindBufferMemory(dc.device, buf, deviceMem, 0))
	if err != nil {
		log.Panicf("Failed to bind device memory to buffer handle")
	}

	return &Buffer{
		handle:    buf,
		deviceMem: deviceMem,
	}
}

type TextureImage struct {
	handle    vk.Image
	deviceMem vk.DeviceMemory
}

func CreateTextureImage(dc *DeviceContext, path string) *TextureImage {
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
	err = vk.Error(vk.MapMemory(dc.device, stgBuf.deviceMem, 0, imgSize, 0, &pData))
	if err != nil {
		log.Panicf("Failed to map device memory")
	}
	vk.Memcopy(pData, img.Pix)
	vk.UnmapMemory(dc.device, stgBuf.deviceMem)

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

func CreateImage(dc *DeviceContext, w uint32, h uint32, format vk.Format, tiling vk.ImageTiling, usage vk.ImageUsageFlags, props vk.MemoryPropertyFlags) (vk.Image, vk.DeviceMemory) {
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
	if vk.CreateImage(dc.device, imageInfo, nil, &img) != vk.Success {
		log.Panicf("failed to create image!")
	}
	memRequirements := readImageMemoryRequirements(dc.device, img)
	allocInfo := &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		PNext:           nil,
		AllocationSize:  memRequirements.Size,
		MemoryTypeIndex: findMemoryType(dc, memRequirements.MemoryTypeBits, props),
	}
	var imgMemory vk.DeviceMemory
	if vk.AllocateMemory(dc.device, allocInfo, nil, &imgMemory) != vk.Success {
		log.Panicf("failed to allocate device memory for image!")
	}
	vk.BindImageMemory(dc.device, img, imgMemory, 0)
	return img, imgMemory
}

func findMemoryType(dc *DeviceContext, typeFilter uint32, propFlags vk.MemoryPropertyFlags) uint32 {
	//log.Printf("Got memory properties: %v", toStringPhysicalDeviceMemProps(c.pdMemoryProps))
	for i := uint32(0); i < dc.pdMemoryProps.MemoryTypeCount; i++ {
		ofType := (typeFilter & (1 << i)) > 0
		hasProperties := dc.pdMemoryProps.MemoryTypes[i].PropertyFlags&propFlags == propFlags
		if ofType && hasProperties {
			log.Printf("Found memory type for buffer -> %d on heap %d", i, dc.pdMemoryProps.MemoryTypes[i].HeapIndex)
			return i
		}
	}
	log.Panicf("Failed to find suitable memory type")
	return 0
}
