package common

import (
	vk "github.com/goki/vulkan"
	"log"
)

type SwapChain struct {
	supDetails SwapChainDetails
	Handle     vk.Swapchain

	Format      vk.SurfaceFormat
	PresentMode vk.PresentMode
	Extend      vk.Extent2D

	Images   []vk.Image
	ImgViews []vk.ImageView
	Aspect   float32

	FrameBuffers []vk.Framebuffer
}

func NewSwapChain(dc *Device, w *Window) *SwapChain {
	sc := &SwapChain{}
	sc.chooseConfiguration(dc, w)
	sc.createSwapChainHandle(dc, w)
	sc.readImages(dc)
	sc.createImageViews(dc)

	// Precalculate the images' aspect ratio for later
	sc.Aspect = float32(sc.Extend.Width) / float32(sc.Extend.Height)

	return sc
}

func (sc *SwapChain) CreateFrameBuffers(dc *Device, renderPass vk.RenderPass, depthImageView *vk.ImageView) {
	sc.FrameBuffers = make([]vk.Framebuffer, len(sc.ImgViews))
	for i := range sc.ImgViews {
		attachments := []vk.ImageView{sc.ImgViews[i]}
		if depthImageView != nil {
			attachments = append(attachments, *depthImageView)
		}
		framebufferInfo := vk.FramebufferCreateInfo{
			SType:           vk.StructureTypeFramebufferCreateInfo,
			PNext:           nil,
			Flags:           0,
			RenderPass:      renderPass,
			AttachmentCount: uint32(len(attachments)),
			PAttachments:    attachments,
			Width:           sc.Extend.Width,
			Height:          sc.Extend.Height,
			Layers:          1,
		}
		fb, err := VkCreateFrameBuffer(dc.D, &framebufferInfo, nil)
		if err != nil {
			log.Panicf("Failed to create frame buffer [%d]", i)
		}
		sc.FrameBuffers[i] = fb
	}
	log.Printf("Successfully created %d frame buffers %v", len(sc.FrameBuffers), sc.FrameBuffers)
}

func (sc *SwapChain) chooseConfiguration(dc *Device, w *Window) {
	sc.supDetails = ReadSwapChainSupportDetails(dc.PD, *w.Surf)
	sc.Format = sc.supDetails.selectSwapSurfaceFormat(vk.FormatB8g8r8a8Srgb, vk.ColorSpaceSrgbNonlinear)
	sc.PresentMode = sc.supDetails.selectSwapPresentMode(vk.PresentModeMailbox)
	sc.Extend = sc.supDetails.selectSwapExtent()
}

func (sc *SwapChain) createSwapChainHandle(dc *Device, w *Window) {
	// Calc reasonable image count for swap chain
	imgCount := sc.supDetails.capabilities.MinImageCount + 1
	imgMaxCount := sc.supDetails.capabilities.MaxImageCount
	if imgCount > 0 && imgCount > imgMaxCount {
		imgCount = imgMaxCount
	}

	// Depending on whether our queue families are the same for graphics and presentation, we need to choose different
	// swap chain configurations: https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain
	indices := dc.QFamilies
	var sharingMode vk.SharingMode
	var indexCount uint32
	qFamIndices := []uint32{*indices.GraphicsFamily, *indices.PresentFamily}
	if *indices.GraphicsFamily != *indices.PresentFamily {
		sharingMode = vk.SharingModeConcurrent
		indexCount = 2
	} else {
		sharingMode = vk.SharingModeExclusive
		indexCount = 0
		qFamIndices = nil
	}

	// Reasonable default values for creating a swap chain
	createInfo := &vk.SwapchainCreateInfo{
		SType:                 vk.StructureTypeSwapchainCreateInfo,
		PNext:                 nil,
		Flags:                 0,
		Surface:               *w.Surf,
		MinImageCount:         imgCount,
		ImageFormat:           sc.Format.Format,
		ImageColorSpace:       sc.Format.ColorSpace,
		ImageExtent:           sc.Extend,
		ImageArrayLayers:      1,
		ImageUsage:            vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit),
		ImageSharingMode:      sharingMode,
		QueueFamilyIndexCount: indexCount,
		PQueueFamilyIndices:   qFamIndices,
		PreTransform:          sc.supDetails.capabilities.CurrentTransform,
		CompositeAlpha:        vk.CompositeAlphaOpaqueBit,
		PresentMode:           sc.PresentMode,
		Clipped:               vk.True,
		OldSwapchain:          nil,
	}

	var err error
	sc.Handle, err = VkCreateSwapChain(dc.D, createInfo, nil)
	if err != nil {
		log.Panicf("Failed create swapchain due to: %s", "err")
	}
	log.Println("Successfully created swap chain")
}

func (sc *SwapChain) readImages(dc *Device) {
	sc.Images = ReadSwapChainImages(dc.D, sc.Handle)
	log.Printf("Read resulting image handles: %v", sc.Images)
}

func (sc *SwapChain) createImageViews(dc *Device) {
	sc.ImgViews = make([]vk.ImageView, len(sc.Images))
	for i := range sc.Images {
		sc.ImgViews[i] = CreateImageViewDC(dc, sc.Images[i], sc.Format.Format, vk.ImageAspectFlags(vk.ImageAspectColorBit))
	}
	log.Printf("Successfully created %d image views %v", len(sc.ImgViews), sc.ImgViews)
}

func (sc *SwapChain) Destroy(dc *Device) {
	for i := range sc.FrameBuffers {
		vk.DestroyFramebuffer(dc.D, sc.FrameBuffers[i], nil)
	}
	for i := range sc.ImgViews {
		vk.DestroyImageView(dc.D, sc.ImgViews[i], nil)
	}
	vk.DestroySwapchain(dc.D, sc.Handle, nil)
}

type SwapChainDetails struct {
	capabilities vk.SurfaceCapabilities
	formats      []vk.SurfaceFormat
	presentModes []vk.PresentMode
}

func (s *SwapChainDetails) selectSwapSurfaceFormat(desiredFormat vk.Format, desiredColorSpace vk.ColorSpace) vk.SurfaceFormat {
	for _, af := range s.formats {
		if af.Format == desiredFormat && af.ColorSpace == desiredColorSpace {
			return af
		}
	}
	fallbackFormat := s.formats[0]
	log.Printf("Did not find prefered SurfaceFormat, selecting first one available. (%v)", fallbackFormat)
	return fallbackFormat
}

func (s *SwapChainDetails) selectSwapPresentMode(desiredMode vk.PresentMode) vk.PresentMode {
	for _, pm := range s.presentModes {
		if pm == desiredMode {
			return pm
		}
	}
	fallbackMode := vk.PresentModeFifo
	log.Printf("Did not find prefered PresentMode, selecting FIFO. (%v)", fallbackMode)
	return fallbackMode
}

func (s *SwapChainDetails) selectSwapExtent() vk.Extent2D {
	// Returning the current extend directly as I dont want to do anything crazy and
	// https://github.com/vulkan-go/demos/blob/master/vulkandraw/vulkandraw.go does the same
	// I can return to this later: https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain
	s.capabilities.CurrentExtent.Deref()
	return s.capabilities.CurrentExtent
}

func checkSwapChainAdequacy(pd vk.PhysicalDevice, surface vk.Surface) bool {
	scDetails := ReadSwapChainSupportDetails(pd, surface)
	log.Printf("Read swap chain details: %v", scDetails)
	return len(scDetails.formats) > 0 && len(scDetails.presentModes) > 0
}

// ToDo: Drop temporarily duplicated code. This should belong into and image wrapping file or allocations
func CreateImageViewDC(dc *Device, image vk.Image, format vk.Format, aspectFlags vk.ImageAspectFlags) vk.ImageView {
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
	imgView, err := VkCreateImageView(dc.D, createInfo, nil)
	if err != nil {
		log.Panicf("Failed create image view due to: %s", err)
	}
	return imgView
}
