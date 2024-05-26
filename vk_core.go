package main

import "C"
import (
	"fmt"
	vk "github.com/goki/vulkan"
	"github.com/veandco/go-sdl2/sdl"
	vm "local/vector_math"
	"log"
	"math"
	"time"
	"unsafe"
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
	pd            vk.PhysicalDevice
	pdMemoryProps vk.PhysicalDeviceMemoryProperties
	device        vk.Device
	qFamilies     QueueFamilyIndices
	graphicsQ     vk.Queue
	presentQ      vk.Queue

	// Target level
	swapChain  vk.Swapchain
	scImages   []vk.Image
	scImgViews []vk.ImageView
	scFormat   vk.SurfaceFormat
	scExtend   vk.Extent2D

	// Drawing infrastructure level
	renderPass          vk.RenderPass
	descriptorSetLayout vk.DescriptorSetLayout
	descriptorPool      vk.DescriptorPool
	descriptorSets      []vk.DescriptorSet
	pipelineLayout      vk.PipelineLayout
	pipelines           []vk.Pipeline
	scFrameBuffers      []vk.Framebuffer
	commandPool         vk.CommandPool

	// Frame level
	commandBuffers     []vk.CommandBuffer
	currentFrameIdx    int32
	imageAvailableSems []vk.Semaphore
	renderFinishedSems []vk.Semaphore
	inFlightFens       []vk.Fence

	// Data level
	vertices             []vm.Vertex
	vertexBuffer         vk.Buffer
	vertexBufferMem      vk.DeviceMemory
	vertIndices          []uint32
	indexBuffer          vk.Buffer
	indexBufferMem       vk.DeviceMemory
	uniformBuffers       []vk.Buffer
	uniformBufferMems    []vk.DeviceMemory
	uniformBuffersMapped []unsafe.Pointer

	// 3D World
	cam  *vm.Camera
	mesh *vm.Mesh
}

// Externally facing functions

func NewRenderCore() *Core {
	w := initSDLWindow()
	initVulkan()
	c := &Core{
		win: w,
	}
	return c
}

func (c *Core) SetScene(m *vm.Mesh, cam *vm.Camera) {
	c.vertices = m.Vertices
	c.vertIndices = m.VIndices
	c.mesh = m
	c.cam = cam
}

func (c *Core) Initialize() {
	c.createInstance()
	c.createSurface()
	c.selectPhysicalDevice()
	c.createLogicalDevice()
	c.createSwapChain()
	c.createImageViews()
	c.createRenderPass()
	c.createDescriptorSetLayout()
	c.createGraphicsPipeline()
	c.createFrameBuffers()
	c.createCommandPool()
	c.createVertexBuffer()
	c.createIndexBuffer()
	c.createUniformBuffers()
	c.createDescriptorPool()
	c.createDescriptorSets()
	c.createCommandBuffers()
	c.createSyncObjects()
}

type iterationHandler func(sdl.Event, *Core)

type drawHandler func(float64, *Core)

// loop this function represents the event-loop for user interaction and currently also contains
// the primary draw call that renders each frame. The whole purpose of this function is to provide
// a neat interface for call backs and all basic functionality a well-behaved app should have. E.g.:
// Not rendering if minimized, close on Window 'close button', close on ESC key.
func (c *Core) loop(ih iterationHandler, dh drawHandler) {
	t0 := time.Now()
	frames := 0
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
			dh(time.Since(t0).Seconds(), c)
			c.drawFrame()
			frames++
		} else {
			// Sleep until new events change c.winMinimized
			sdl.WaitEvent()
		}
	}
	t1 := time.Now()
	dtSec := float64(t1.Sub(t0).Milliseconds()) / 1000
	log.Printf("Elapsed: %vs, rough avg fps: %v fps", dtSec, float64(frames)/dtSec)
}

func (c *Core) destroy() {
	// We need to wait for the last asynchronous call to finish before tear down
	vk.DeviceWaitIdle(c.device)
	c.destroySwapChainAndDerivatives()

	// Destroy all buffers (application data)
	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		vk.DestroyBuffer(c.device, c.uniformBuffers[i], nil)
		vk.FreeMemory(c.device, c.uniformBufferMems[i], nil)
	}
	vk.DestroyDescriptorPool(c.device, c.descriptorPool, nil)
	vk.DestroyDescriptorSetLayout(c.device, c.descriptorSetLayout, nil)
	vk.DestroyBuffer(c.device, c.vertexBuffer, nil)
	vk.FreeMemory(c.device, c.vertexBufferMem, nil)
	vk.DestroyBuffer(c.device, c.indexBuffer, nil)
	vk.FreeMemory(c.device, c.indexBufferMem, nil)

	// Destroy all infrastructure up to the sdl window
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
	err := c.win.Destroy()
	if err != nil {
		log.Fatal(err)
	}
}

func (c *Core) destroySwapChainAndDerivatives() {
	for i := range c.scFrameBuffers {
		vk.DestroyFramebuffer(c.device, c.scFrameBuffers[i], nil)
	}
	for i := range c.scImgViews {
		vk.DestroyImageView(c.device, c.scImgViews[i], nil)
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
	ins, err := VkCreateInstance(createInfo, nil)
	if err != nil {
		log.Panicf("Failed to create vk instance, due to: %v", err)
	}
	c.instance = ins
}

func (c *Core) createSurface() {
	surf, err := sdlCreateVkSurface(c.win, c.instance)
	if err != nil {
		log.Panicf("Failed to create SDL window's Vulkan-surface, due to: %v", err)
	}
	c.surface = surf
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
	}
	log.Printf("Found suitable device")
	c.pd = pd

	// Also set related member variables for c.pd as they are needed later
	qf, err := findQueueFamilies(c.pd, c.surface)
	if err != nil {
		log.Panicf("Failed to read queue families from selected device due to: %s", err)
	}
	c.qFamilies = *qf
	c.pdMemoryProps = readDeviceMemoryProperties(c.pd)
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
	queueInfos := c.qFamilies.toQueueCreateInfos()
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

	var err error
	c.device, err = VkCreateDevice(c.pd, deviceCreatInfo, nil)
	if err != nil {
		log.Panicf("Failed create logical device due to: %s", "err")
	}
	c.graphicsQ, err = VkGetDeviceQueue(c.device, c.qFamilies.graphicsFamily, 0)
	if err != nil {
		log.Panicf("Failed to get 'graphics' device queue: %s", err)
	}
	c.presentQ, err = VkGetDeviceQueue(c.device, c.qFamilies.presentFamily, 0)
	if err != nil {
		log.Panicf("Failed to get 'present' device queue: %s", err)
	}
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
	indices := c.qFamilies
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

	var err error
	c.swapChain, err = VkCreateSwapChain(c.device, createInfo, nil)
	if err != nil {
		log.Panicf("Failed create swapchain due to: %s", "err")
	}
	log.Println("Successfully created swap chain")

	c.scImages = readSwapChainImages(c.device, c.swapChain)
	log.Printf("Read resulting image handles: %v", c.scImages)
}

func (c *Core) createImageViews() {
	c.scImgViews = make([]vk.ImageView, len(c.scImages))
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
		var err error
		c.scImgViews[i], err = VkCreateImageView(c.device, createInfo, nil)
		if err != nil {
			log.Panicf("Failed create image view %d due to: %s", i, "err")
		}
	}
	log.Printf("Successfully created %d image views %v", len(c.scImgViews), c.scImgViews)
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
	var err error
	c.renderPass, err = VkCreateRenderPass(c.device, &renderPassInfo, nil)
	if err != nil {
		log.Panicf("Failed create render pass due to: %s", "err")
	}
	log.Println("Successfully created render pass")
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
	bindingDesc := []vk.VertexInputBindingDescription{vm.GetVertexBindingDescription()}
	attributeDesc := vm.GetVertexAttributeDescriptions()
	vertexInputInfo := vk.PipelineVertexInputStateCreateInfo{
		SType:                           vk.StructureTypePipelineVertexInputStateCreateInfo,
		PNext:                           nil,
		Flags:                           0,
		VertexBindingDescriptionCount:   1,
		PVertexBindingDescriptions:      bindingDesc,
		VertexAttributeDescriptionCount: uint32(len(attributeDesc)),
		PVertexAttributeDescriptions:    attributeDesc,
	}
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
	viewportStateInfo := vk.PipelineViewportStateCreateInfo{
		SType:         vk.StructureTypePipelineViewportStateCreateInfo,
		PNext:         nil,
		Flags:         0,
		ViewportCount: 1,
		PViewports:    nil,
		ScissorCount:  1,
		PScissors:     nil,
	}
	rasterizerInfo := vk.PipelineRasterizationStateCreateInfo{
		SType:                   vk.StructureTypePipelineRasterizationStateCreateInfo,
		PNext:                   nil,
		Flags:                   0,
		DepthClampEnable:        vk.False,
		RasterizerDiscardEnable: vk.False,
		PolygonMode:             vk.PolygonModeFill,
		CullMode:                vk.CullModeFlags(vk.CullModeBackBit),
		FrontFace:               vk.FrontFaceCounterClockwise,
		DepthBiasEnable:         vk.False,
		DepthBiasConstantFactor: 0,
		DepthBiasClamp:          0,
		DepthBiasSlopeFactor:    0,
		LineWidth:               1.0,
	}
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
		SetLayoutCount:         1,
		PSetLayouts:            []vk.DescriptorSetLayout{c.descriptorSetLayout},
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
	c.scFrameBuffers = make([]vk.Framebuffer, len(c.scImgViews))
	for i := range c.scImgViews {
		framebufferInfo := vk.FramebufferCreateInfo{
			SType:           vk.StructureTypeFramebufferCreateInfo,
			PNext:           nil,
			Flags:           0,
			RenderPass:      c.renderPass,
			AttachmentCount: 1,
			PAttachments:    []vk.ImageView{c.scImgViews[i]},
			Width:           c.scExtend.Width,
			Height:          c.scExtend.Height,
			Layers:          1,
		}
		var err error
		c.scFrameBuffers[i], err = VkCreateFrameBuffer(c.device, &framebufferInfo, nil)
		if err != nil {
			log.Panicf("Failed to create frame buffer [%d]", i)
		}
	}
	log.Printf("Successfully created %d frame buffers %v", len(c.scFrameBuffers), c.scFrameBuffers)
}

func (c *Core) createCommandPool() {
	poolInfo := vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		PNext:            nil,
		Flags:            vk.CommandPoolCreateFlags(vk.CommandPoolCreateResetCommandBufferBit),
		QueueFamilyIndex: *c.qFamilies.graphicsFamily,
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

func (c *Core) createBuffer(size vk.DeviceSize, usage vk.BufferUsageFlags, memProperties vk.MemoryPropertyFlags) (vk.Buffer, vk.DeviceMemory) {
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
	err := vk.Error(vk.CreateBuffer(c.device, &bufferInfo, nil, &buf))
	if err != nil {
		log.Panicf("Failed to create vertex buffer")
	}
	bufRequirements := readBufferMemoryRequirements(c.device, buf)

	// Allocate device memory
	allocInfo := vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		PNext:           nil,
		AllocationSize:  bufRequirements.Size,
		MemoryTypeIndex: c.findMemoryType(bufRequirements.MemoryTypeBits, memProperties),
	}
	var deviceMem vk.DeviceMemory
	err = vk.Error(vk.AllocateMemory(c.device, &allocInfo, nil, &deviceMem))
	if err != nil {
		log.Panicf("Failed to allocate vertex buffer memory")
	}

	// Associate allocated memory with buffer handle
	err = vk.Error(vk.BindBufferMemory(c.device, buf, deviceMem, 0))
	if err != nil {
		log.Panicf("Failed to bind device memory to buffer handle")
	}

	return buf, deviceMem
}

func (c *Core) createVertexBuffer() {
	// Create staging buffer
	bufSize := vk.DeviceSize(int(unsafe.Sizeof(c.vertices[0])) * len(c.vertices))
	stagingBuffer, stagingBufferMem := c.createBuffer(
		bufSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
	)

	// Map staging memory - copy our vertex data into staging - unmap staging again
	var pData unsafe.Pointer
	err := vk.Error(vk.MapMemory(c.device, stagingBufferMem, 0, bufSize, 0, &pData))
	if err != nil {
		log.Panicf("Failed to map device memory")
	}
	vk.Memcopy(pData, rawBytes(c.vertices))
	vk.UnmapMemory(c.device, stagingBufferMem)

	// Create vertex buffer
	c.vertexBuffer, c.vertexBufferMem = c.createBuffer(
		bufSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferDstBit|vk.BufferUsageVertexBufferBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit),
	)
	log.Printf(
		"Created vertex buffer handle@%v -> memAddr@%v -> Size: %d Byte",
		c.vertexBuffer, c.vertexBufferMem, bufSize,
	)

	// Move memory to vertex buffer & delete staging buffer afterwards
	c.copyBuffer(stagingBuffer, c.vertexBuffer, bufSize)
	vk.DestroyBuffer(c.device, stagingBuffer, nil)
	vk.FreeMemory(c.device, stagingBufferMem, nil)
}

func (c *Core) createIndexBuffer() {
	log.Printf("Indices: %v", c.vertIndices)
	bufSize := vk.DeviceSize(int(unsafe.Sizeof(c.vertIndices[0])) * len(c.vertIndices))
	log.Printf("Buffer size: %dByte", bufSize)
	stagingBuffer, stagingBufferMem := c.createBuffer(
		bufSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
	)

	// Map staging memory - copy our vertex data into staging - unmap staging again
	var pData unsafe.Pointer
	err := vk.Error(vk.MapMemory(c.device, stagingBufferMem, 0, bufSize, 0, &pData))
	if err != nil {
		log.Panicf("Failed to map device memory")
	}
	vk.Memcopy(pData, rawBytes(c.vertIndices))
	vk.UnmapMemory(c.device, stagingBufferMem)

	// Create vertex buffer
	c.indexBuffer, c.indexBufferMem = c.createBuffer(
		bufSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferDstBit|vk.BufferUsageIndexBufferBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit),
	)

	// Move memory to vertex buffer & delete staging buffer afterwards
	c.copyBuffer(stagingBuffer, c.indexBuffer, bufSize)
	vk.DestroyBuffer(c.device, stagingBuffer, nil)
	vk.FreeMemory(c.device, stagingBufferMem, nil)
}

// copyBuffer is a subroutine that prepares a command buffer that is then executed on the device.
// The command buffer is allocated, records the copy command and is submitted to the device. After idle
// the command buffer is freed.
func (c *Core) copyBuffer(src vk.Buffer, dst vk.Buffer, s vk.DeviceSize) {
	allocInfo := vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		PNext:              nil,
		CommandPool:        c.commandPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}
	cmdBuffers := make([]vk.CommandBuffer, 1)
	vk.AllocateCommandBuffers(c.device, &allocInfo, cmdBuffers)

	beginInfo := vk.CommandBufferBeginInfo{
		SType:            vk.StructureTypeCommandBufferBeginInfo,
		PNext:            nil,
		Flags:            vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit),
		PInheritanceInfo: nil,
	}
	vk.BeginCommandBuffer(cmdBuffers[0], &beginInfo)
	copyRegions := []vk.BufferCopy{
		{
			SrcOffset: 0,
			DstOffset: 0,
			Size:      s,
		},
	}
	vk.CmdCopyBuffer(cmdBuffers[0], src, dst, 1, copyRegions)
	vk.EndCommandBuffer(cmdBuffers[0])

	submitInfo := vk.SubmitInfo{
		SType:                vk.StructureTypeSubmitInfo,
		PNext:                nil,
		WaitSemaphoreCount:   0,
		PWaitSemaphores:      nil,
		PWaitDstStageMask:    nil,
		CommandBufferCount:   1,
		PCommandBuffers:      cmdBuffers,
		SignalSemaphoreCount: 0,
		PSignalSemaphores:    nil,
	}
	vk.QueueSubmit(c.graphicsQ, 1, []vk.SubmitInfo{submitInfo}, nil)
	vk.QueueWaitIdle(c.graphicsQ)
	vk.FreeCommandBuffers(c.device, c.commandPool, 1, cmdBuffers)
}

func (c *Core) findMemoryType(typeFilter uint32, propFlags vk.MemoryPropertyFlags) uint32 {
	//log.Printf("Got memory properties: %v", toStringPhysicalDeviceMemProps(c.pdMemoryProps))
	for i := uint32(0); i < c.pdMemoryProps.MemoryTypeCount; i++ {
		ofType := (typeFilter & (1 << i)) > 0
		hasProperties := c.pdMemoryProps.MemoryTypes[i].PropertyFlags&propFlags == propFlags
		if ofType && hasProperties {
			log.Printf("Found memory type for buffer -> %d on heap %d", i, c.pdMemoryProps.MemoryTypes[i].HeapIndex)
			return i
		}
	}
	log.Panicf("Failed to find suitable memory type")
	return 0
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
		vk.NewClearValue([]float32{0.01, 0.01, 0.01, 1}),
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

	vertBuffers := []vk.Buffer{c.vertexBuffer}
	offsets := []vk.DeviceSize{0}
	vk.CmdBindVertexBuffers(buffer, 0, uint32(len(vertBuffers)), vertBuffers, offsets)
	vk.CmdBindIndexBuffer(buffer, c.indexBuffer, 0, vk.IndexTypeUint32)
	vk.CmdBindDescriptorSets(buffer, vk.PipelineBindPointGraphics, c.pipelineLayout, 0, 1, []vk.DescriptorSet{c.descriptorSets[imageIdx]}, 0, nil)

	vk.CmdDrawIndexed(buffer, uint32(len(c.vertIndices)), 1, 0, 0, 0)

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

	c.updateUniformBuffer(c.currentFrameIdx)

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

func (c *Core) createDescriptorSetLayout() {
	uboLayoutBinding := vk.DescriptorSetLayoutBinding{
		Binding:            0,                              // <- binding index in vert shader
		DescriptorType:     vk.DescriptorTypeUniformBuffer, // <- type of binding in vert shader
		DescriptorCount:    1,
		StageFlags:         vk.ShaderStageFlags(vk.ShaderStageVertexBit),
		PImmutableSamplers: nil,
	}
	layoutInfo := vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		PNext:        nil,
		Flags:        0,
		BindingCount: 1,
		PBindings:    []vk.DescriptorSetLayoutBinding{uboLayoutBinding},
	}
	var dsl vk.DescriptorSetLayout
	if vk.CreateDescriptorSetLayout(c.device, &layoutInfo, nil, &dsl) != vk.Success {
		log.Panicf("Failed to create descriptor set layout")
	}
	c.descriptorSetLayout = dsl
}

func (c *Core) createUniformBuffers() {
	uboBufSize := vk.DeviceSize(int(SizeOfUbo()))
	log.Printf("UBO buffer size: %d Byte", uboBufSize)

	c.uniformBuffers = make([]vk.Buffer, MAX_FRAMES_IN_FLIGHT)
	c.uniformBufferMems = make([]vk.DeviceMemory, MAX_FRAMES_IN_FLIGHT)
	c.uniformBuffersMapped = make([]unsafe.Pointer, MAX_FRAMES_IN_FLIGHT)

	memProps := vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit | vk.MemoryPropertyHostCoherentBit)
	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		c.uniformBuffers[i], c.uniformBufferMems[i] = c.createBuffer(uboBufSize, vk.BufferUsageFlags(vk.BufferUsageUniformBufferBit), memProps)
		vk.MapMemory(c.device, c.uniformBufferMems[i], 0, uboBufSize, 0, &c.uniformBuffersMapped[i])
	}
}

func (c *Core) createDescriptorPool() {
	poolSize := vk.DescriptorPoolSize{
		Type:            vk.DescriptorTypeUniformBuffer,
		DescriptorCount: MAX_FRAMES_IN_FLIGHT,
	}
	poolInfo := vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		PNext:         nil,
		Flags:         0,
		MaxSets:       MAX_FRAMES_IN_FLIGHT,
		PoolSizeCount: 1,
		PPoolSizes:    []vk.DescriptorPoolSize{poolSize},
	}
	var dp vk.DescriptorPool
	if vk.CreateDescriptorPool(c.device, &poolInfo, nil, &dp) != vk.Success {
		log.Panicf("Failed to create descriptor pool")
	}
	c.descriptorPool = dp
}

func (c *Core) createDescriptorSets() {
	layouts := []vk.DescriptorSetLayout{c.descriptorSetLayout, c.descriptorSetLayout, c.descriptorSetLayout}
	allocInfo := vk.DescriptorSetAllocateInfo{
		SType:              vk.StructureTypeDescriptorSetAllocateInfo,
		PNext:              nil,
		DescriptorPool:     c.descriptorPool,
		DescriptorSetCount: MAX_FRAMES_IN_FLIGHT,
		PSetLayouts:        layouts,
	}
	sets := make([]vk.DescriptorSet, MAX_FRAMES_IN_FLIGHT)
	if vk.AllocateDescriptorSets(c.device, &allocInfo, &(sets[0])) != vk.Success {
		log.Panicf("Failed to allocate descriptor set")
	}
	log.Printf("%v", sets)
	c.descriptorSets = sets

	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		bufferInfo := vk.DescriptorBufferInfo{
			Buffer: c.uniformBuffers[i],
			Offset: 0,
			Range:  vk.DeviceSize(SizeOfUbo()),
		}
		descriptorWrite := vk.WriteDescriptorSet{
			SType:            vk.StructureTypeWriteDescriptorSet,
			PNext:            nil,
			DstSet:           c.descriptorSets[i],
			DstBinding:       0,
			DstArrayElement:  0,
			DescriptorCount:  1,
			DescriptorType:   vk.DescriptorTypeUniformBuffer,
			PImageInfo:       nil,
			PBufferInfo:      []vk.DescriptorBufferInfo{bufferInfo},
			PTexelBufferView: nil,
		}
		vk.UpdateDescriptorSets(c.device, 1, []vk.WriteDescriptorSet{descriptorWrite}, 0, nil)
	}
}

func (c *Core) updateUniformBuffer(frameIdx int32) {
	c.cam.Aspect = float32(c.scExtend.Width) / float32(c.scExtend.Height)
	ubo := UniformBufferObject{
		model:      c.mesh.ModelMat,
		view:       c.cam.GetView(),
		projection: c.cam.GetProjection(),
	}
	vk.Memcopy(c.uniformBuffersMapped[frameIdx], ubo.Bytes())
}
