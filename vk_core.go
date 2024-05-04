package main

import (
	"fmt"
	vk "github.com/goki/vulkan"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"math"
)

type Core struct {
	// OS/Window level
	win          *sdl.Window
	winResized   bool
	winMinimized bool
	winClose     bool
	instance     vk.Instance
	surface      vk.Surface

	// Device level
	pd        vk.PhysicalDevice
	device    vk.Device
	graphicsQ vk.Queue
	presentQ  vk.Queue

	// Window-Device interoperability
	swapChain vk.Swapchain
	scImages  []vk.Image
	scFormat  vk.SurfaceFormat
	scExtend  vk.Extent2D

	// Drawing infrastructure
	imgViews       []vk.ImageView
	renderPass     vk.RenderPass
	pipelineLayout vk.PipelineLayout
	pipelines      []vk.Pipeline
	scFrameBuffers []vk.Framebuffer
	commandPool    vk.CommandPool

	// Draw / Frame level
	commandBuffers     []vk.CommandBuffer
	currentFrameIdx    int32
	imageAvailableSems []vk.Semaphore
	renderFinishedSems []vk.Semaphore
	inFlightFens       []vk.Fence
}

// Externally facing functions

func NewRenderCore() *Core {
	w := initSDLWindow()
	initVulkan()
	c := &Core{
		win: w,
	}
	c.initialize()
	return c
}

func (c *Core) initialize() {
	c.createInstance()
	c.createSurface()
	c.selectPhysicalDevice()
	c.createLogicalDevice()
	c.createSwapChain()
	c.createImageViews()
	c.createRenderPass()
	c.createGraphicsPipeline()
	c.createFrameBuffers()
	c.createCommandPool()
	c.createCommandBuffers()
	c.createSyncObjects()
}

type iterationHandler func(sdl.Event, *Core)

func (c *Core) loop(ih iterationHandler) {
	var event sdl.Event
	c.winClose = false
	for !c.winClose {
		for event = sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			// Doing some basic functionality for basic window handling
			switch ev := event.(type) {
			case *sdl.QuitEvent:
				c.winClose = true
			case *sdl.WindowEvent:
				if ev.Event == sdl.WINDOWEVENT_RESIZED {
					c.winResized = true
				} else if ev.Event == sdl.WINDOWEVENT_MINIMIZED {
					c.winMinimized = true
				} else if ev.Event == sdl.WINDOWEVENT_RESTORED {
					c.winMinimized = false
				}
			case *sdl.KeyboardEvent:
				if ev.Keysym.Sym == sdl.K_ESCAPE {
					c.winClose = true
				}
			}
			ih(event, c)
		}
		if !c.winMinimized {
			c.drawFrame()
		} else {
			// Sleep until new events change c.winMinimized
			sdl.WaitEvent()
		}
	}
}

func (c *Core) destroy() {
	// We need to wait for the last asynchronous call to finish before tear down
	vk.DeviceWaitIdle(c.device)
	c.destroySwapChainAndDerivatives()

	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		vk.DestroySemaphore(c.device, c.imageAvailableSems[i], nil)
		vk.DestroySemaphore(c.device, c.renderFinishedSems[i], nil)
		vk.DestroyFence(c.device, c.inFlightFens[i], nil)
	}
	vk.DestroyCommandPool(c.device, c.commandPool, nil)

	for i := range c.pipelines {
		vk.DestroyPipeline(c.device, c.pipelines[i], nil)
	}
	vk.DestroyPipelineLayout(c.device, c.pipelineLayout, nil)
	vk.DestroyRenderPass(c.device, c.renderPass, nil)

	vk.DestroySurface(c.instance, c.surface, nil)
	vk.DestroyDevice(c.device, nil)
	vk.DestroyInstance(c.instance, nil)
	c.win.Destroy()
}

func (c *Core) destroySwapChainAndDerivatives() {
	for i := range c.scFrameBuffers {
		vk.DestroyFramebuffer(c.device, c.scFrameBuffers[i], nil)
	}
	for i := range c.imgViews {
		vk.DestroyImageView(c.device, c.imgViews[i], nil)
	}
	vk.DestroySwapchain(c.device, c.swapChain, nil)
}

// Bootstrapping / Initialization code

func initSDLWindow() *sdl.Window {
	// SDL - version print & init
	sdlVersion := fmt.Sprintf("v%d.%d.%d", sdl.MAJOR_VERSION, sdl.MINOR_VERSION, sdl.PATCHLEVEL)
	log.Printf("Using SDL: [%s]", sdlVersion)
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic("Failed to initialize a SDL !")
	}
	log.Println("Creating SDL window")
	win, err := sdl.CreateWindow(
		PROGRAM_NAME,
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		WINDOW_WIDTH,
		WINDOW_HEIGHT,
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE|sdl.WINDOW_VULKAN,
	)
	if err != nil {
		panic(err)
	}
	return win
}

func initVulkan() {
	// Vulkan - spec as per: https://github.com/goki/vulkan
	log.Printf("Vulkan SDK: [%s]", "v1.3.239")
	log.Println("Initializing Vulkan for SDL window")
	vk.SetGetInstanceProcAddr(sdl.VulkanGetVkGetInstanceProcAddr())
	err := vk.Init()
	if err != nil {
		panic(err)
	}
}

func (c *Core) createInstance() {
	requiredExtensions := c.win.VulkanGetInstanceExtensions()
	checkInstanceExtensionSupport(requiredExtensions)

	if ENABLE_VALIDATION {
		log.Printf("Validation enabled, checking layer support")
		checkValidationLayerSupport(VALIDATION_LAYERS)
	}
	var instance vk.Instance
	applicationInfo := &vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		PNext:              nil,
		PApplicationName:   "GPU fluid simulation",
		ApplicationVersion: vk.MakeVersion(1, 0, 0),
		PEngineName:        "No Engine",
		EngineVersion:      vk.MakeVersion(1, 0, 0),
		ApiVersion:         vk.MakeVersion(1, 0, 0),
	}
	createInfo := &vk.InstanceCreateInfo{
		SType:                   vk.StructureTypeInstanceCreateInfo,
		PNext:                   nil,
		Flags:                   0,
		PApplicationInfo:        applicationInfo,
		EnabledLayerCount:       0,
		PpEnabledLayerNames:     nil,
		EnabledExtensionCount:   uint32(len(requiredExtensions)),
		PpEnabledExtensionNames: terminatedStrs(requiredExtensions),
	}
	if ENABLE_VALIDATION {
		createInfo.EnabledLayerCount = uint32(len(VALIDATION_LAYERS))
		createInfo.PpEnabledLayerNames = terminatedStrs(VALIDATION_LAYERS)
	}
	creationResult := vk.CreateInstance(createInfo, nil, &instance)
	if creationResult != vk.Success {
		log.Panicf("Failed to create vk instance, result: %v", creationResult)
	}
	err := vk.InitInstance(instance)
	if err != nil {
		log.Panicf("Failed to init instance with %s", err)
	}
	c.instance = instance
}

func (c *Core) createSurface() {
	surfPtr, err := c.win.VulkanCreateSurface(c.instance)
	if err != nil {
		log.Panicf("Failed to create SDL window Vulkan surface due to: %s", err)
	}
	c.surface = vk.SurfaceFromPointer(uintptr(surfPtr))
}

func (c *Core) selectPhysicalDevice() {
	availableDevices := readPhysicalDevices(c.instance)
	var pd vk.PhysicalDevice
	for i := range availableDevices {
		if isDeviceSuitable(availableDevices[i], c.surface) {
			pd = availableDevices[i]
			break
		}
	}
	if pd == nil {
		log.Panicf("No suitable physical device (GPU) found")
	} else {
		log.Printf("Found suitable device")
	}
	c.pd = pd
}

func isDeviceSuitable(pd vk.PhysicalDevice, surface vk.Surface) bool {
	pdProps := readPhysicalDeviceProperties(pd)
	pdFeatures := readPhysicalDeviceFeatures(pd)
	pdQueueFams := readQueueFamilies(pd)

	log.Printf("Physical divece\n%s", toStringPhysicalDeviceTable(pdProps, pdFeatures, pdQueueFams))

	indices, err := findQueueFamilies(pd, surface)
	if err != nil {
		log.Printf("Failed to get required queue families: %s", err)
		return false
	}

	queuesSupported := indices.isAllQueuesFound()
	isDiscreteGPU := pdProps.DeviceType == vk.PhysicalDeviceTypeDiscreteGpu
	featuresSupported := pdFeatures.GeometryShader == 1
	extensionsSupported := checkDeviceExtensionSupport(pd, DEVICE_EXTENSIONS)

	isSwapChainAdequate := false
	if extensionsSupported {
		isSwapChainAdequate = checkSwapChainAdequacy(pd, surface)
	}

	return isDiscreteGPU && featuresSupported && queuesSupported && extensionsSupported && isSwapChainAdequate
}

func (c *Core) createLogicalDevice() {
	qFamilies, _ := findQueueFamilies(c.pd, c.surface)
	queueInfos := qFamilies.toQueueCreateInfos()
	deviceFeatures := vk.PhysicalDeviceFeatures{} // Empty for now as we dont need anything special at the moment
	deviceCreatInfo := &vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		PNext:                   nil,
		Flags:                   0,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PQueueCreateInfos:       queueInfos,
		EnabledLayerCount:       0,
		PpEnabledLayerNames:     nil,
		EnabledExtensionCount:   uint32(len(DEVICE_EXTENSIONS)),
		PpEnabledExtensionNames: terminatedStrs(DEVICE_EXTENSIONS),
		PEnabledFeatures:        []vk.PhysicalDeviceFeatures{deviceFeatures},
	}
	if ENABLE_VALIDATION {
		deviceCreatInfo.EnabledLayerCount = uint32(len(VALIDATION_LAYERS))
		deviceCreatInfo.PpEnabledLayerNames = terminatedStrs(VALIDATION_LAYERS)
	}
	var d vk.Device
	if vk.CreateDevice(c.pd, deviceCreatInfo, nil, &d) != vk.Success {
		log.Panicf("Failed create logical device due to: %s", "err")
	}
	c.device = d

	var gq vk.Queue
	gfIndex, err := qFamilies.graphicsFamilyIdx()
	if err != nil {
		log.Panicf("Failed to access graphics capable queue family index: %s", err)
	}
	vk.GetDeviceQueue(c.device, gfIndex, 0, &gq)
	log.Printf("Retrived graphics queue handle: %v", gq)
	c.graphicsQ = gq

	var pq vk.Queue
	presentIndex, err := qFamilies.presentFamilyIdx()
	if err != nil {
		log.Panicf("Failed to access graphics capable queue family index: %s", err)
	}
	vk.GetDeviceQueue(c.device, presentIndex, 0, &pq)
	log.Printf("Retrived present queue handle: %v", pq)
	c.presentQ = pq
}

func (c *Core) createSwapChain() {
	scDetails := readSwapChainSupportDetails(c.pd, c.surface)
	c.scFormat = scDetails.selectSwapSurfaceFormat()
	scPresentMode := scDetails.selectSwapPresentMode()
	c.scExtend = scDetails.chooseSwapExtent()

	imgCount := scDetails.capabilities.MinImageCount + 1
	imgMaxCount := scDetails.capabilities.MaxImageCount
	if imgCount > 0 && imgCount > imgMaxCount {
		imgCount = imgMaxCount
	}

	// Depending on whether our queue families are the same for graphics and presentation, we need to choose different
	// swap chain configurations: https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain
	indices, err := findQueueFamilies(c.pd, c.surface)
	if err != nil {
		log.Panicf("Failed to read queue families when creating swap chain: %s", err)
	}
	var sharingMode vk.SharingMode
	var indexCount uint32
	qFamIndices := []uint32{*indices.graphicsFamily, *indices.presentFamily}
	if *indices.graphicsFamily != *indices.presentFamily {
		sharingMode = vk.SharingModeConcurrent
		indexCount = 2
	} else {
		sharingMode = vk.SharingModeExclusive
		indexCount = 0
		qFamIndices = nil
	}

	createInfo := &vk.SwapchainCreateInfo{
		SType:                 vk.StructureTypeSwapchainCreateInfo,
		PNext:                 nil,
		Flags:                 0,
		Surface:               c.surface,
		MinImageCount:         imgCount,
		ImageFormat:           c.scFormat.Format,
		ImageColorSpace:       c.scFormat.ColorSpace,
		ImageExtent:           c.scExtend,
		ImageArrayLayers:      1,
		ImageUsage:            vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit),
		ImageSharingMode:      sharingMode,
		QueueFamilyIndexCount: indexCount,
		PQueueFamilyIndices:   qFamIndices,
		PreTransform:          scDetails.capabilities.CurrentTransform,
		CompositeAlpha:        vk.CompositeAlphaOpaqueBit,
		PresentMode:           scPresentMode,
		Clipped:               vk.True,
		OldSwapchain:          nil,
	}

	var sc vk.Swapchain
	if vk.CreateSwapchain(c.device, createInfo, nil, &sc) != vk.Success {
		log.Panicf("Failed create swapchain due to: %s", "err")
	}
	log.Println("Successfully created swap chain")
	c.swapChain = sc
	c.scImages = readSwapChainImages(c.device, c.swapChain)
	log.Printf("Read resulting image handles: %v", c.scImages)
}

func (c *Core) createImageViews() {
	imgViews := make([]vk.ImageView, len(c.scImages))
	for i := range c.scImages {
		createInfo := &vk.ImageViewCreateInfo{
			SType:    vk.StructureTypeImageViewCreateInfo,
			PNext:    nil,
			Flags:    0,
			Image:    c.scImages[i],
			ViewType: vk.ImageViewType2d,
			Format:   c.scFormat.Format,
			Components: vk.ComponentMapping{
				R: vk.ComponentSwizzleIdentity,
				G: vk.ComponentSwizzleIdentity,
				B: vk.ComponentSwizzleIdentity,
				A: vk.ComponentSwizzleIdentity,
			},
			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask:     vk.ImageAspectFlags(vk.ImageAspectColorBit),
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		}
		if vk.CreateImageView(c.device, createInfo, nil, &imgViews[i]) != vk.Success {
			log.Panicf("Failed create image view %d due to: %s", i, "err")
		}
	}
	c.imgViews = imgViews
	log.Printf("Successfully created %d image views", len(c.imgViews))
}

func (c *Core) createRenderPass() {
	colorAttachment := vk.AttachmentDescription{
		Flags:          0,
		Format:         c.scFormat.Format,
		Samples:        vk.SampleCount1Bit,
		LoadOp:         vk.AttachmentLoadOpClear,
		StoreOp:        vk.AttachmentStoreOpStore,
		StencilLoadOp:  vk.AttachmentLoadOpDontCare,
		StencilStoreOp: vk.AttachmentStoreOpDontCare,
		InitialLayout:  vk.ImageLayoutUndefined,
		FinalLayout:    vk.ImageLayoutPresentSrc,
	}

	colorAttachmentRef := vk.AttachmentReference{
		Attachment: 0,
		Layout:     vk.ImageLayoutColorAttachmentOptimal,
	}
	subpass := vk.SubpassDescription{
		Flags:                   0,
		PipelineBindPoint:       vk.PipelineBindPointGraphics,
		InputAttachmentCount:    0,
		PInputAttachments:       nil,
		ColorAttachmentCount:    1,
		PColorAttachments:       []vk.AttachmentReference{colorAttachmentRef},
		PResolveAttachments:     nil,
		PDepthStencilAttachment: nil,
		PreserveAttachmentCount: 0,
		PPreserveAttachments:    nil,
	}

	dependency := vk.SubpassDependency{
		SrcSubpass:      vk.SubpassExternal,
		DstSubpass:      0,
		SrcStageMask:    vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		DstStageMask:    vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		SrcAccessMask:   0,
		DstAccessMask:   vk.AccessFlags(vk.AccessColorAttachmentWriteBit),
		DependencyFlags: 0,
	}

	renderPassInfo := vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		PNext:           nil,
		Flags:           0,
		AttachmentCount: 1,
		PAttachments:    []vk.AttachmentDescription{colorAttachment},
		SubpassCount:    1,
		PSubpasses:      []vk.SubpassDescription{subpass},
		DependencyCount: 1,
		PDependencies:   []vk.SubpassDependency{dependency},
	}
	var renderPass vk.RenderPass
	if vk.CreateRenderPass(c.device, &renderPassInfo, nil, &renderPass) != vk.Success {
		log.Panicf("Failed create render pass due to: %s", "err")
	}
	log.Println("Successfully created render pass")
	c.renderPass = renderPass
}

func (c *Core) createGraphicsPipeline() {
	vertShaderMod := readShaderCode(c.device, "shaders_spv/vert.spv")
	log.Printf("Loaded vertex shader: %v", vertShaderMod)
	fragShaderMod := readShaderCode(c.device, "shaders_spv/frag.spv")
	log.Printf("Loaded fragment shader: %v", fragShaderMod)

	vertexShaderStageInfo := vk.PipelineShaderStageCreateInfo{
		SType:               vk.StructureTypePipelineShaderStageCreateInfo,
		PNext:               nil,
		Flags:               0,
		Stage:               vk.ShaderStageVertexBit,
		Module:              vertShaderMod,
		PName:               "main\x00", // entrypoint -> function name in the shader
		PSpecializationInfo: nil,
	}

	fragmentShaderStageInfo := vk.PipelineShaderStageCreateInfo{
		SType:               vk.StructureTypePipelineShaderStageCreateInfo,
		PNext:               nil,
		Flags:               0,
		Stage:               vk.ShaderStageFragmentBit,
		Module:              fragShaderMod,
		PName:               "main\x00", // entrypoint -> function name in the shader
		PSpecializationInfo: nil,
	}

	shaderStages := []vk.PipelineShaderStageCreateInfo{vertexShaderStageInfo, fragmentShaderStageInfo}
	log.Printf("Prepared %d shader stages for pipeline creation: %v", len(shaderStages), shaderStages)

	// Dynamic state
	dynamicStates := []vk.DynamicState{
		vk.DynamicStateViewport,
		vk.DynamicStateScissor,
	}
	dynamicStateCreateInfo := vk.PipelineDynamicStateCreateInfo{
		SType:             vk.StructureTypePipelineDynamicStateCreateInfo,
		PNext:             nil,
		Flags:             0,
		DynamicStateCount: uint32(len(dynamicStates)),
		PDynamicStates:    dynamicStates,
	}
	log.Printf("PipelineDynamicStateCreateInfo: %v", dynamicStateCreateInfo)

	// Vertex input - as we are not passing in any vertex data at the moment
	// (hard coded in the shader), we dont add any real info here
	vertexInputInfo := vk.PipelineVertexInputStateCreateInfo{
		SType:                           vk.StructureTypePipelineVertexInputStateCreateInfo,
		PNext:                           nil,
		Flags:                           0,
		VertexBindingDescriptionCount:   0,
		PVertexBindingDescriptions:      nil,
		VertexAttributeDescriptionCount: 0,
		PVertexAttributeDescriptions:    nil,
	}
	log.Printf("PipelineVertexInputStateCreateInfo: %v", vertexInputInfo)

	// Input assembly - this is how the vertices are "put together" and allows us to do optimizations on what
	// data is passed to the GPU. Its interesting, but we will stick to the tutorial for now. See:
	// https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Fixed_functions for more.
	inputAssemblyInfo := vk.PipelineInputAssemblyStateCreateInfo{
		SType:                  vk.StructureTypePipelineInputAssemblyStateCreateInfo,
		PNext:                  nil,
		Flags:                  0,
		Topology:               vk.PrimitiveTopologyTriangleList,
		PrimitiveRestartEnable: vk.False,
	}
	log.Printf("PipelineInputAssemblyStateCreateInfo: %v", inputAssemblyInfo)

	viewportStateInfo := vk.PipelineViewportStateCreateInfo{
		SType:         vk.StructureTypePipelineViewportStateCreateInfo,
		PNext:         nil,
		Flags:         0,
		ViewportCount: 1,
		PViewports:    nil,
		ScissorCount:  1,
		PScissors:     nil,
	}
	log.Printf("PipelineViewportStateCreateInfo: %v", viewportStateInfo)

	rasterizerInfo := vk.PipelineRasterizationStateCreateInfo{
		SType:                   vk.StructureTypePipelineRasterizationStateCreateInfo,
		PNext:                   nil,
		Flags:                   0,
		DepthClampEnable:        vk.False,
		RasterizerDiscardEnable: vk.False,
		PolygonMode:             vk.PolygonModeFill,
		CullMode:                vk.CullModeFlags(vk.CullModeBackBit),
		FrontFace:               vk.FrontFaceClockwise,
		DepthBiasEnable:         vk.False,
		DepthBiasConstantFactor: 0,
		DepthBiasClamp:          0,
		DepthBiasSlopeFactor:    0,
		LineWidth:               1.0,
	}
	log.Printf("PipelineRasterizationStateCreateInfo: %v", rasterizerInfo)

	multisamplingInfo := vk.PipelineMultisampleStateCreateInfo{
		SType:                 vk.StructureTypePipelineMultisampleStateCreateInfo,
		PNext:                 nil,
		Flags:                 0,
		RasterizationSamples:  vk.SampleCount1Bit,
		SampleShadingEnable:   vk.False,
		MinSampleShading:      1.0,
		PSampleMask:           nil,
		AlphaToCoverageEnable: vk.False,
		AlphaToOneEnable:      vk.False,
	}
	log.Printf("PipelineMultisampleStateCreateInfo: %v", multisamplingInfo)

	colorBlendAttachmentInfo := vk.PipelineColorBlendAttachmentState{
		BlendEnable:         vk.False,
		SrcColorBlendFactor: vk.BlendFactorOne,
		DstColorBlendFactor: vk.BlendFactorZero,
		ColorBlendOp:        vk.BlendOpAdd,
		SrcAlphaBlendFactor: vk.BlendFactorOne,
		DstAlphaBlendFactor: vk.BlendFactorZero,
		AlphaBlendOp:        vk.BlendOpAdd,
		ColorWriteMask:      vk.ColorComponentFlags(vk.ColorComponentRBit | vk.ColorComponentGBit | vk.ColorComponentBBit | vk.ColorComponentABit),
	}

	colorBlendingInfo := vk.PipelineColorBlendStateCreateInfo{
		SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
		PNext:           nil,
		Flags:           0,
		LogicOpEnable:   vk.False,
		LogicOp:         vk.LogicOpCopy,
		AttachmentCount: 1,
		PAttachments:    []vk.PipelineColorBlendAttachmentState{colorBlendAttachmentInfo},
		BlendConstants:  [4]float32{0, 0, 0, 0},
	}
	log.Printf("PipelineColorBlendStateCreateInfo: %v", colorBlendingInfo)

	// Pipeline layouts are used to pass uniforms as they will be specified during pipeline creation
	var pipelineLayout vk.PipelineLayout
	pipelineLayoutInfo := vk.PipelineLayoutCreateInfo{
		SType:                  vk.StructureTypePipelineLayoutCreateInfo,
		PNext:                  nil,
		Flags:                  0,
		SetLayoutCount:         0,
		PSetLayouts:            nil,
		PushConstantRangeCount: 0,
		PPushConstantRanges:    nil,
	}
	if vk.CreatePipelineLayout(c.device, &pipelineLayoutInfo, nil, &pipelineLayout) != vk.Success {
		log.Panicf("Failed to create pipeline layout")
	}
	c.pipelineLayout = pipelineLayout
	log.Printf("PipelineLayout: %v", pipelineLayout)

	// The actual pipeline
	pipelineInfo := vk.GraphicsPipelineCreateInfo{
		SType:               vk.StructureTypeGraphicsPipelineCreateInfo,
		PNext:               nil,
		Flags:               0,
		StageCount:          2,
		PStages:             shaderStages,
		PVertexInputState:   &vertexInputInfo,
		PInputAssemblyState: &inputAssemblyInfo,
		PTessellationState:  nil,
		PViewportState:      &viewportStateInfo,
		PRasterizationState: &rasterizerInfo,
		PMultisampleState:   &multisamplingInfo,
		PDepthStencilState:  nil,
		PColorBlendState:    &colorBlendingInfo,
		PDynamicState:       &dynamicStateCreateInfo,
		Layout:              pipelineLayout,
		RenderPass:          c.renderPass,
		Subpass:             0,
		BasePipelineHandle:  nil,
		BasePipelineIndex:   -1,
	}
	pipelineInfos := []vk.GraphicsPipelineCreateInfo{pipelineInfo}
	var graphicsPipelines [1]vk.Pipeline // <- how do I allocate this correctly
	if vk.CreateGraphicsPipelines(c.device, nil, 1, pipelineInfos, nil, graphicsPipelines[:]) != vk.Success {
		log.Panicf("Failed to create graphics pipeline")
	}
	log.Printf("Successfully created graphics pipeline")
	c.pipelines = graphicsPipelines[:]

	// As shader modules are just c thin wrapper to bring the code over to the GPU, the modules can be disposed of
	// immediately at the end of this function.
	vk.DestroyShaderModule(c.device, vertShaderMod, nil)
	vk.DestroyShaderModule(c.device, fragShaderMod, nil)
}

func (c *Core) createFrameBuffers() {
	c.scFrameBuffers = make([]vk.Framebuffer, len(c.imgViews))
	for i := range c.imgViews {
		attachments := []vk.ImageView{c.imgViews[i]}
		framebufferInfo := vk.FramebufferCreateInfo{
			SType:           vk.StructureTypeFramebufferCreateInfo,
			PNext:           nil,
			Flags:           0,
			RenderPass:      c.renderPass,
			AttachmentCount: 1,
			PAttachments:    attachments,
			Width:           c.scExtend.Width,
			Height:          c.scExtend.Height,
			Layers:          1,
		}
		if vk.CreateFramebuffer(c.device, &framebufferInfo, nil, &c.scFrameBuffers[i]) != vk.Success {
			log.Panicf("Failed to create frame buffer [%d]", i)
		}
		log.Printf("Successfully created frame buffer [%d]", i)
	}
}

func (c *Core) createCommandPool() {
	indices, err := findQueueFamilies(c.pd, c.surface)
	if err != nil {
		log.Panicf("Failed to find queue families when creating command pool due to: %v", err)
	}
	poolInfo := vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		PNext:            nil,
		Flags:            vk.CommandPoolCreateFlags(vk.CommandPoolCreateResetCommandBufferBit),
		QueueFamilyIndex: *indices.graphicsFamily,
	}
	var commandPool vk.CommandPool
	if vk.CreateCommandPool(c.device, &poolInfo, nil, &commandPool) != vk.Success {
		log.Panicf("Failed to create command pool")
	}
	log.Printf("Successfully created command pool")
	c.commandPool = commandPool
}

func (c *Core) createCommandBuffers() {
	var buffers = make([]vk.CommandBuffer, MAX_FRAMES_IN_FLIGHT)
	cbAllocateInfo := vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		PNext:              nil,
		CommandPool:        c.commandPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: uint32(len(buffers)),
	}

	if vk.AllocateCommandBuffers(c.device, &cbAllocateInfo, buffers) != vk.Success {
		log.Panicf("Failed to allocate command buffers")
	}
	log.Printf("Successfully allocated %d command buffers", len(buffers))
	c.commandBuffers = buffers
}

func (c *Core) createSyncObjects() {
	ias := make([]vk.Semaphore, MAX_FRAMES_IN_FLIGHT)
	rfs := make([]vk.Semaphore, MAX_FRAMES_IN_FLIGHT)
	iff := make([]vk.Fence, MAX_FRAMES_IN_FLIGHT)
	semCreateInfo := vk.SemaphoreCreateInfo{
		SType: vk.StructureTypeSemaphoreCreateInfo,
		PNext: nil,
		Flags: 0,
	}
	fenCreateInfo := vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
		PNext: nil,
		Flags: vk.FenceCreateFlags(vk.FenceCreateSignaledBit),
	}
	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		if vk.CreateSemaphore(c.device, &semCreateInfo, nil, &ias[i]) != vk.Success ||
			vk.CreateSemaphore(c.device, &semCreateInfo, nil, &rfs[i]) != vk.Success ||
			vk.CreateFence(c.device, &fenCreateInfo, nil, &iff[i]) != vk.Success {
			log.Panicf("Failed tocreate sync objects")
		}
	}
	c.imageAvailableSems = ias
	c.renderFinishedSems = rfs
	c.inFlightFens = iff
}

// Drawing and derivative functionality

func (c *Core) recordCommandBuffer(buffer vk.CommandBuffer, imageIdx uint32) {
	// Begin recording
	beginInfo := vk.CommandBufferBeginInfo{
		SType:            vk.StructureTypeCommandBufferBeginInfo,
		PNext:            nil,
		Flags:            0,
		PInheritanceInfo: nil,
	}
	if vk.BeginCommandBuffer(buffer, &beginInfo) != vk.Success {
		log.Panicf("Failed to begin recording command buffer")
	}

	// Start render pass
	renderArea := vk.Rect2D{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: c.scExtend,
	}
	clearValues := []vk.ClearValue{
		vk.NewClearValue([]float32{0, 0, 0, 1}),
	}
	renderPassInfo := vk.RenderPassBeginInfo{
		SType:           vk.StructureTypeRenderPassBeginInfo,
		PNext:           nil,
		RenderPass:      c.renderPass,
		Framebuffer:     c.scFrameBuffers[imageIdx],
		RenderArea:      renderArea,
		ClearValueCount: uint32(len(clearValues)),
		PClearValues:    clearValues,
	}
	vk.CmdBeginRenderPass(buffer, &renderPassInfo, vk.SubpassContentsInline)

	vk.CmdBindPipeline(buffer, vk.PipelineBindPointGraphics, c.pipelines[0])

	viewport := []vk.Viewport{
		{
			X:        0,
			Y:        0,
			Width:    float32(c.scExtend.Width),
			Height:   float32(c.scExtend.Height),
			MinDepth: 0,
			MaxDepth: 1.0,
		},
	}
	vk.CmdSetViewport(buffer, 0, 1, viewport)

	scissor := []vk.Rect2D{
		{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: c.scExtend,
		},
	}
	vk.CmdSetScissor(buffer, 0, 1, scissor)

	vk.CmdDraw(buffer, 3, 1, 0, 0)
	vk.CmdEndRenderPass(buffer)
	if vk.EndCommandBuffer(buffer) != vk.Success {
		log.Printf("Failed to record commandbuffer")
	}
}

func (c *Core) drawFrame() {
	// Wait for frame to be ready - signalled by the inFlightFens
	vk.WaitForFences(c.device, 1, []vk.Fence{c.inFlightFens[c.currentFrameIdx]}, vk.True, math.MaxUint64)

	var imgIdx uint32
	result := vk.AcquireNextImage(c.device, c.swapChain, math.MaxUint64, c.imageAvailableSems[c.currentFrameIdx], nil, &imgIdx)
	// React on surface changes and other possible causes for failure (e.g.: Window resizing)
	if result == vk.ErrorOutOfDate {
		c.recreateSwapChain()
		return
	} else if result != vk.Success && result != vk.Suboptimal {
		log.Panicf("Failed to aquire image, AcquireNextImage(...) result code: %d", result)
	}

	// Reset the fence only if we are actually going to execute work that will put the fence into the signalled state
	vk.ResetFences(c.device, 1, []vk.Fence{c.inFlightFens[c.currentFrameIdx]})

	vk.ResetCommandBuffer(c.commandBuffers[c.currentFrameIdx], 0)
	c.recordCommandBuffer(c.commandBuffers[c.currentFrameIdx], imgIdx)

	submitInfo := vk.SubmitInfo{
		SType:              vk.StructureTypeSubmitInfo,
		PNext:              nil,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{c.imageAvailableSems[c.currentFrameIdx]},
		PWaitDstStageMask: []vk.PipelineStageFlags{
			vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		},
		CommandBufferCount:   1,
		PCommandBuffers:      []vk.CommandBuffer{c.commandBuffers[c.currentFrameIdx]},
		SignalSemaphoreCount: 1,
		PSignalSemaphores:    []vk.Semaphore{c.renderFinishedSems[c.currentFrameIdx]},
	}
	if vk.QueueSubmit(c.graphicsQ, 1, []vk.SubmitInfo{submitInfo}, c.inFlightFens[c.currentFrameIdx]) != vk.Success {
		log.Panicf("Failed to submit commandbuffer")
	}

	presentInfo := vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		PNext:              nil,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{c.renderFinishedSems[c.currentFrameIdx]},
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{c.swapChain},
		PImageIndices:      []uint32{imgIdx},
		PResults:           nil,
	}
	result = vk.QueuePresent(c.presentQ, &presentInfo)
	// React on surface changes and other possible causes for failure (e.g.: Window resizing)
	if result == vk.ErrorOutOfDate || result == vk.Suboptimal || c.winResized {
		c.winResized = false
		c.recreateSwapChain()
	} else if result != vk.Success {
		log.Panicf("Failed to present image, QueuePresent(...) result code: %d", result)
	}

	c.currentFrameIdx = (c.currentFrameIdx + 1) % MAX_FRAMES_IN_FLIGHT
}

func (c *Core) recreateSwapChain() {
	vk.DeviceWaitIdle(c.device)
	c.destroySwapChainAndDerivatives()
	c.createSwapChain()
	c.createImageViews()
	c.createFrameBuffers()
}
