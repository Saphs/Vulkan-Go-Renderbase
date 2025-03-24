package renderer

import "C"
import (
	com "GPU_fluid_simulation/common"
	"GPU_fluid_simulation/model"
	"fmt"
	vk "github.com/goki/vulkan"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"math"
	"neilpa.me/go-stbi"
	"time"
	"unsafe"
)

const PROGRAM_NAME = "GPU fluid simulation"
const WINDOW_WIDTH, WINDOW_HEIGHT int32 = 1280, 720
const MAX_FRAMES_IN_FLIGHT = 3

type Core struct {
	// OS/Window level
	win    *com.Window
	device *com.Device

	// Target level
	swapChain *com.SwapChain

	// Drawing infrastructure level
	renderPass          vk.RenderPass
	descriptorSetLayout vk.DescriptorSetLayout
	descriptorPool      vk.DescriptorPool
	descriptorSets      []vk.DescriptorSet
	pipelineLayout      vk.PipelineLayout
	pipelines           []vk.Pipeline
	commandPool         vk.CommandPool

	// DepthBuffer

	// Frame level
	commandBuffers     []vk.CommandBuffer
	currentFrameIdx    int32
	imageAvailableSems []vk.Semaphore
	renderFinishedSems []vk.Semaphore
	inFlightFens       []vk.Fence

	// Data level
	uniformBuffers       []vk.Buffer
	uniformBufferMems    []vk.DeviceMemory
	uniformBuffersMapped []unsafe.Pointer

	// 3D World
	Cam    *model.Camera
	models []*model.Model

	textureImage     vk.Image
	textureImageMem  vk.DeviceMemory
	textureImageView vk.ImageView
	textureSampler   vk.Sampler

	depthImage     vk.Image
	depthImageMem  vk.DeviceMemory
	depthImageView vk.ImageView
}

// Externally facing functions

func NewRenderCore() *Core {
	c := &Core{}
	c.Initialize()
	return c
}

func (c *Core) Initialize() {
	c.win = com.NewWindow(PROGRAM_NAME, WINDOW_WIDTH, WINDOW_HEIGHT, []string{
		"VK_LAYER_KHRONOS_validation",
	})
	c.device = com.NewDevice(c.win)
	c.swapChain = com.NewSwapChain(c.device, c.win)
	c.createRenderPass()
	c.createDescriptorSetLayout()
	c.createGraphicsPipeline()
	c.createCommandPool()
	c.createDepthResources()
	c.createFrameBuffers()
	c.createTexture()
	c.createTextureViews()
	c.createTextureSampler()
	c.createUniformBuffers()
	c.createDescriptorPool()
	c.createDescriptorSets()
	c.createCommandBuffers()
	c.createSyncObjects()
}

type iterationHandler func(sdl.Event, *Core)

type drawHandler func(time.Duration, *Core)

// Loop this function represents the event-loop for user interaction and currently also contains
// the primary draw call that renders each frame. The whole purpose of this function is to provide
// a neat interface for call backs and all basic functionality a well-behaved app should have. E.g.:
// Not rendering if minimized, close on Window 'close button', close on ESC key.
func (c *Core) Loop(ih iterationHandler, dh drawHandler) {
	t0 := time.Now()
	frames := 0
	var event sdl.Event
	c.win.Close = false
	for !c.win.Close {
		for event = sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			// Doing some basic functionality for basic window handling
			switch ev := event.(type) {
			case *sdl.QuitEvent:
				c.win.Close = true
			case *sdl.WindowEvent:
				if ev.Event == sdl.WINDOWEVENT_RESIZED {
					c.win.Resized = true
				} else if ev.Event == sdl.WINDOWEVENT_MINIMIZED {
					c.win.Minimized = true
				} else if ev.Event == sdl.WINDOWEVENT_RESTORED {
					c.win.Minimized = false
				}
			case *sdl.KeyboardEvent:
				if ev.Keysym.Sym == sdl.K_ESCAPE {
					c.win.Close = true
				}
			}
			ih(event, c)
		}
		if !c.win.Minimized {
			dh(time.Since(t0), c)
			c.drawFrame()
			frames++
		} else {
			// Sleep until new events change c.winMinimized
			sdl.WaitEvent()
		}
	}
	dt := time.Since(t0)
	log.Printf("Elapsed: %v, rough avg fps: %v fps", dt, float64(frames)/dt.Seconds())
}

func (c *Core) Destroy() {
	// If user has not cleaned up all models manually, warn and remove them now
	if len(c.models) > 0 {
		log.Printf("Leftover models in render core!: %v", len(c.models))
		c.ClearScene()
	}

	// We need to wait for the last asynchronous call to finish before tear down
	vk.DeviceWaitIdle(c.device.D)
	c.destroySwapChainAndDerivatives()

	vk.DestroySampler(c.device.D, c.textureSampler, nil)
	vk.DestroyImageView(c.device.D, c.textureImageView, nil)
	vk.DestroyImage(c.device.D, c.textureImage, nil)
	vk.FreeMemory(c.device.D, c.textureImageMem, nil)

	// Destroy all buffers (application data)
	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		vk.DestroyBuffer(c.device.D, c.uniformBuffers[i], nil)
		vk.FreeMemory(c.device.D, c.uniformBufferMems[i], nil)
	}
	vk.DestroyDescriptorPool(c.device.D, c.descriptorPool, nil)
	vk.DestroyDescriptorSetLayout(c.device.D, c.descriptorSetLayout, nil)

	// Destroy all infrastructure up to the sdl window
	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		vk.DestroySemaphore(c.device.D, c.imageAvailableSems[i], nil)
		vk.DestroySemaphore(c.device.D, c.renderFinishedSems[i], nil)
		vk.DestroyFence(c.device.D, c.inFlightFens[i], nil)
	}
	vk.DestroyCommandPool(c.device.D, c.commandPool, nil)

	for i := range c.pipelines {
		vk.DestroyPipeline(c.device.D, c.pipelines[i], nil)
	}
	vk.DestroyPipelineLayout(c.device.D, c.pipelineLayout, nil)
	vk.DestroyRenderPass(c.device.D, c.renderPass, nil)

	c.device.Destroy()
	c.win.Destroy()
}

func (c *Core) destroySwapChainAndDerivatives() {
	vk.DestroyImageView(c.device.D, c.depthImageView, nil)
	vk.DestroyImage(c.device.D, c.depthImage, nil)
	vk.FreeMemory(c.device.D, c.depthImageMem, nil)

	c.swapChain.Destroy(c.device)
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

func (c *Core) createImageView(image vk.Image, format vk.Format, aspectFlags vk.ImageAspectFlags) vk.ImageView {
	return CreateImageViewDC(c.device, image, format, aspectFlags)
}

func CreateImageViewDC(dc *com.Device, image vk.Image, format vk.Format, aspectFlags vk.ImageAspectFlags) vk.ImageView {
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
	imgView, err := com.VkCreateImageView(dc.D, createInfo, nil)
	if err != nil {
		log.Panicf("Failed create image view due to: %s", err)
	}
	return imgView
}

func (c *Core) createRenderPass() {
	colorAttachment := vk.AttachmentDescription{
		Flags:          0,
		Format:         c.swapChain.Format.Format,
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
	depthAttachment := vk.AttachmentDescription{
		Flags:          0,
		Format:         c.findDepthFormat(),
		Samples:        vk.SampleCount1Bit,
		LoadOp:         vk.AttachmentLoadOpClear,
		StoreOp:        vk.AttachmentStoreOpDontCare,
		StencilLoadOp:  vk.AttachmentLoadOpDontCare,
		StencilStoreOp: vk.AttachmentStoreOpDontCare,
		InitialLayout:  vk.ImageLayoutUndefined,
		FinalLayout:    vk.ImageLayoutDepthStencilAttachmentOptimal,
	}
	depthAttachmentRef := vk.AttachmentReference{
		Attachment: 1,
		Layout:     vk.ImageLayoutDepthStencilAttachmentOptimal,
	}
	subpass := vk.SubpassDescription{
		Flags:                   0,
		PipelineBindPoint:       vk.PipelineBindPointGraphics,
		InputAttachmentCount:    0,
		PInputAttachments:       nil,
		ColorAttachmentCount:    1,
		PColorAttachments:       []vk.AttachmentReference{colorAttachmentRef},
		PResolveAttachments:     nil,
		PDepthStencilAttachment: &depthAttachmentRef,
		PreserveAttachmentCount: 0,
		PPreserveAttachments:    nil,
	}
	dependency := vk.SubpassDependency{
		SrcSubpass:      vk.SubpassExternal,
		DstSubpass:      0,
		SrcStageMask:    vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit | vk.PipelineStageEarlyFragmentTestsBit),
		DstStageMask:    vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit | vk.PipelineStageEarlyFragmentTestsBit),
		SrcAccessMask:   0,
		DstAccessMask:   vk.AccessFlags(vk.AccessColorAttachmentWriteBit | vk.AccessDepthStencilAttachmentWriteBit),
		DependencyFlags: 0,
	}
	renderPassInfo := vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		PNext:           nil,
		Flags:           0,
		AttachmentCount: 2,
		PAttachments:    []vk.AttachmentDescription{colorAttachment, depthAttachment},
		SubpassCount:    1,
		PSubpasses:      []vk.SubpassDescription{subpass},
		DependencyCount: 1,
		PDependencies:   []vk.SubpassDependency{dependency},
	}
	var err error
	c.renderPass, err = com.VkCreateRenderPass(c.device.D, &renderPassInfo, nil)
	if err != nil {
		log.Panicf("Failed create render pass due to: %s", "err")
	}
	log.Println("Successfully created render pass")
}

func (c *Core) createGraphicsPipeline() {
	vertShaderMod, vertStageInfo := LoadVert(c.device.D, "shaders_spv/vert.spv")
	fragShaderMod, fragStageInfo := LoadFrag(c.device.D, "shaders_spv/frag.spv")
	shaderStages := []vk.PipelineShaderStageCreateInfo{vertStageInfo, fragStageInfo}
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
	bindingDesc := []vk.VertexInputBindingDescription{model.GetVertexBindingDescription()}

	attributeDesc := model.GetVertexAttributeDescriptions()
	log.Printf("attributeDesc: %v", attributeDesc)
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
	modelPushConstantRange := vk.PushConstantRange{
		StageFlags: vk.ShaderStageFlags(vk.ShaderStageVertexBit),
		Offset:     0,
		Size:       model.ModelPushConstantsSize(),
	}
	pipelineLayoutInfo := vk.PipelineLayoutCreateInfo{
		SType:                  vk.StructureTypePipelineLayoutCreateInfo,
		PNext:                  nil,
		Flags:                  0,
		SetLayoutCount:         1,
		PSetLayouts:            []vk.DescriptorSetLayout{c.descriptorSetLayout},
		PushConstantRangeCount: 1,
		PPushConstantRanges:    []vk.PushConstantRange{modelPushConstantRange},
	}
	layouts, err := com.VkCreatePipelineLayout(c.device.D, &pipelineLayoutInfo, nil)
	if err != nil {
		log.Panicf("Failed to create pipeline layout")
	}
	c.pipelineLayout = layouts

	depthStencil := vk.PipelineDepthStencilStateCreateInfo{
		SType:                 vk.StructureTypePipelineDepthStencilStateCreateInfo,
		PNext:                 nil,
		Flags:                 0,
		DepthTestEnable:       vk.True,
		DepthWriteEnable:      vk.True,
		DepthCompareOp:        vk.CompareOpLess,
		DepthBoundsTestEnable: vk.False,
		StencilTestEnable:     vk.False,
		Front:                 vk.StencilOpState{},
		Back:                  vk.StencilOpState{},
		MinDepthBounds:        0,
		MaxDepthBounds:        1,
	}

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
		PDepthStencilState:  &depthStencil,
		PColorBlendState:    &colorBlendingInfo,
		PDynamicState:       &dynamicStateCreateInfo,
		Layout:              c.pipelineLayout,
		RenderPass:          c.renderPass,
		Subpass:             0,
		BasePipelineHandle:  nil,
		BasePipelineIndex:   -1,
	}
	pipelineInfos := []vk.GraphicsPipelineCreateInfo{pipelineInfo}
	pipelines, err := com.VkCreateGraphicsPipelines(c.device.D, nil, 1, pipelineInfos, nil)
	if err != nil {
		log.Panicf("Failed to create graphics pipeline")
	}
	c.pipelines = pipelines
	log.Printf("Successfully created graphics pipeline")

	// Explicitly done right after pipeline creation
	DeleteShaderMod(c.device.D, vertShaderMod)
	DeleteShaderMod(c.device.D, fragShaderMod)
}

func (c *Core) createFrameBuffers() {
	c.swapChain.CreateFrameBuffers(c.device, c.renderPass, &c.depthImageView)
}

func (c *Core) createCommandPool() {
	poolInfo := vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		PNext:            nil,
		Flags:            vk.CommandPoolCreateFlags(vk.CommandPoolCreateResetCommandBufferBit),
		QueueFamilyIndex: *c.device.QFamilies.GraphicsFamily,
	}
	commandPool, err := com.VkCreateCommandPool(c.device.D, &poolInfo, nil)
	if err != nil {
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

	if vk.AllocateCommandBuffers(c.device.D, &cbAllocateInfo, buffers) != vk.Success {
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
		if vk.CreateSemaphore(c.device.D, &semCreateInfo, nil, &ias[i]) != vk.Success ||
			vk.CreateSemaphore(c.device.D, &semCreateInfo, nil, &rfs[i]) != vk.Success ||
			vk.CreateFence(c.device.D, &fenCreateInfo, nil, &iff[i]) != vk.Success {
			log.Panicf("Failed tocreate sync objects")
		}
	}
	c.imageAvailableSems = ias
	c.renderFinishedSems = rfs
	c.inFlightFens = iff
}

func (c *Core) allocateVBuffer(m *model.Model) (vk.Buffer, vk.DeviceMemory) {
	// Create staging buffer
	bufSize := vk.DeviceSize(m.GetVBufferSize())
	stgBuf := com.CreateBuffer(
		c.device,
		bufSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
	)

	// Copy our vertex data into staging (device) memory
	com.CopyToDeviceBuffer(c.device, stgBuf, m.GetVBufferBytes())

	// Create vertex buffer
	vertBuf := com.CreateBuffer(
		c.device,
		bufSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferDstBit|vk.BufferUsageVertexBufferBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit),
	)
	log.Printf(
		"Created vertex buffer (\"%s\": [handleRef@%p, bufferRef@%p, Size: %d Byte])",
		m.Name, &vertBuf.Handle, &vertBuf.DeviceMem, bufSize,
	)

	// Move memory to vertex buffer & delete staging buffer afterwards
	c.copyBuffer(stgBuf, vertBuf, bufSize)
	com.DestroyBuffer(c.device, stgBuf)

	return vertBuf.Handle, vertBuf.DeviceMem
}

func (c *Core) allocateIdxBuffer(m *model.Model) (vk.Buffer, vk.DeviceMemory) {
	// Create staging buffer
	bufSize := vk.DeviceSize(m.GetIdxBufferSize())
	stgBuf := com.CreateBuffer(
		c.device,
		bufSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
	)

	// Map staging memory - copy our vertex data into staging - unmap staging again
	var pData unsafe.Pointer
	err := vk.Error(vk.MapMemory(c.device.D, stgBuf.DeviceMem, 0, bufSize, 0, &pData))
	if err != nil {
		log.Panicf("Failed to map device memory")
	}
	vk.Memcopy(pData, com.RawBytes(m.Mesh.VIndices))
	vk.UnmapMemory(c.device.D, stgBuf.DeviceMem)

	// Create vertex buffer
	idxBuf := com.CreateBuffer(
		c.device,
		bufSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferDstBit|vk.BufferUsageIndexBufferBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit),
	)
	log.Printf(
		"Created index buffer (\"%s\": [handleRef@%p, bufferRef@%p, Size: %d Byte])",
		m.Name, &idxBuf.Handle, &idxBuf.DeviceMem, bufSize,
	)

	// Move memory to vertex buffer & delete staging buffer afterwards
	c.copyBuffer(stgBuf, idxBuf, bufSize)
	vk.DestroyBuffer(c.device.D, stgBuf.Handle, nil)
	vk.FreeMemory(c.device.D, stgBuf.DeviceMem, nil)

	return idxBuf.Handle, idxBuf.DeviceMem
}

func (c *Core) copyBuffer(src *com.Buffer, dst *com.Buffer, s vk.DeviceSize) {
	c.copyVkBuffer(src.Handle, dst.Handle, s)
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
	c.endSingleTimeCommands(cmdBuf)
}

func (c *Core) transitionImageLayout(img vk.Image, format vk.Format, old vk.ImageLayout, new vk.ImageLayout) {
	cmdBuf := c.beginSingleTimeCommands()

	var aspectFlags vk.ImageAspectFlags
	if new == vk.ImageLayoutDepthStencilAttachmentOptimal {
		aspectFlags = vk.ImageAspectFlags(vk.ImageAspectDepthBit)
		if hasStencilComponent(format) {
			aspectFlags = vk.ImageAspectFlags(vk.ImageAspectDepthBit | vk.ImageAspectStencilBit)
		}
	} else {
		aspectFlags = vk.ImageAspectFlags(vk.ImageAspectColorBit)
	}

	var srcStage vk.PipelineStageFlags
	var dstStage vk.PipelineStageFlags
	barrier := vk.ImageMemoryBarrier{
		SType:               vk.StructureTypeImageMemoryBarrier,
		PNext:               nil,
		SrcAccessMask:       0, // set below
		DstAccessMask:       0, // set below
		OldLayout:           old,
		NewLayout:           new,
		SrcQueueFamilyIndex: vk.QueueFamilyIgnored,
		DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		Image:               img,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     aspectFlags,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}

	if old == vk.ImageLayoutUndefined && new == vk.ImageLayoutTransferDstOptimal {
		barrier.SrcAccessMask = 0
		barrier.DstAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)
		srcStage = vk.PipelineStageFlags(vk.PipelineStageTopOfPipeBit)
		dstStage = vk.PipelineStageFlags(vk.PipelineStageTransferBit)
	} else if old == vk.ImageLayoutTransferDstOptimal && new == vk.ImageLayoutShaderReadOnlyOptimal {
		barrier.SrcAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)
		barrier.DstAccessMask = vk.AccessFlags(vk.AccessShaderReadBit)
		srcStage = vk.PipelineStageFlags(vk.PipelineStageTransferBit)
		dstStage = vk.PipelineStageFlags(vk.PipelineStageFragmentShaderBit)
	} else if old == vk.ImageLayoutUndefined && new == vk.ImageLayoutDepthStencilAttachmentOptimal {
		barrier.SrcAccessMask = 0
		barrier.DstAccessMask = vk.AccessFlags(vk.AccessDepthStencilAttachmentReadBit | vk.AccessDepthStencilAttachmentWriteBit)
		srcStage = vk.PipelineStageFlags(vk.PipelineStageTopOfPipeBit)
		dstStage = vk.PipelineStageFlags(vk.PipelineStageEarlyFragmentTestsBit)
	} else {
		log.Panicf("unsupported image layout transition!")
	}

	vk.CmdPipelineBarrier(
		cmdBuf,
		srcStage, dstStage,
		0,
		0, nil,
		0, nil,
		1, []vk.ImageMemoryBarrier{barrier},
	)

	c.endSingleTimeCommands(cmdBuf)
}

func (c *Core) copyBufferToImage(buffer vk.Buffer, img vk.Image, w uint32, h uint32) {
	cmdBuf := c.beginSingleTimeCommands()
	region := vk.BufferImageCopy{
		BufferOffset:      0,
		BufferRowLength:   0,
		BufferImageHeight: 0,
		ImageSubresource: vk.ImageSubresourceLayers{
			AspectMask:     vk.ImageAspectFlags(vk.ImageAspectColorBit),
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		ImageOffset: vk.Offset3D{
			X: 0,
			Y: 0,
			Z: 0,
		},
		ImageExtent: vk.Extent3D{
			Width:  w,
			Height: h,
			Depth:  1,
		},
	}
	vk.CmdCopyBufferToImage(cmdBuf, buffer, img, vk.ImageLayoutTransferDstOptimal, 1, []vk.BufferImageCopy{region})
	c.endSingleTimeCommands(cmdBuf)
}

func (c *Core) beginSingleTimeCommands() vk.CommandBuffer {
	allocInfo := vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		PNext:              nil,
		CommandPool:        c.commandPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}
	cmdBuffers := make([]vk.CommandBuffer, 1)
	vk.AllocateCommandBuffers(c.device.D, &allocInfo, cmdBuffers)

	beginInfo := vk.CommandBufferBeginInfo{
		SType:            vk.StructureTypeCommandBufferBeginInfo,
		PNext:            nil,
		Flags:            vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit),
		PInheritanceInfo: nil,
	}
	vk.BeginCommandBuffer(cmdBuffers[0], &beginInfo)
	return cmdBuffers[0]
}

func (c *Core) endSingleTimeCommands(cmdBuf vk.CommandBuffer) {
	vk.EndCommandBuffer(cmdBuf)

	submitInfo := vk.SubmitInfo{
		SType:                vk.StructureTypeSubmitInfo,
		PNext:                nil,
		WaitSemaphoreCount:   0,
		PWaitSemaphores:      nil,
		PWaitDstStageMask:    nil,
		CommandBufferCount:   1,
		PCommandBuffers:      []vk.CommandBuffer{cmdBuf},
		SignalSemaphoreCount: 0,
		PSignalSemaphores:    nil,
	}
	vk.QueueSubmit(c.device.GraphicsQ, 1, []vk.SubmitInfo{submitInfo}, nil)
	vk.QueueWaitIdle(c.device.GraphicsQ)
	vk.FreeCommandBuffers(c.device.D, c.commandPool, 1, []vk.CommandBuffer{cmdBuf})
}

func (c *Core) createTexture() {
	path := "textures/facebook.jpg"
	img, err := stbi.Load(path)
	if err != nil {
		log.Panicf("Failed to load %s: %v", path, err)
	}
	w := img.Rect.Dx()
	h := img.Rect.Dy()
	bytesPerPixel := 4
	imgSize := vk.DeviceSize(w * h * bytesPerPixel)
	log.Printf("Loaded image %s (w: %dp, h:%d) %d Byte", path, w, h, imgSize)

	stgBuf := com.CreateBuffer(
		c.device,
		imgSize,
		vk.BufferUsageFlags(vk.BufferUsageTransferSrcBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit),
	)
	// Map staging memory - copy our vertex data into staging - unmap staging again
	var pData unsafe.Pointer
	err = vk.Error(vk.MapMemory(c.device.D, stgBuf.DeviceMem, 0, imgSize, 0, &pData))
	if err != nil {
		log.Panicf("Failed to map device memory")
	}
	vk.Memcopy(pData, img.Pix)
	vk.UnmapMemory(c.device.D, stgBuf.DeviceMem)

	c.textureImage, c.textureImageMem = com.CreateImage(
		c.device,
		uint32(w),
		uint32(h),
		vk.FormatR8g8b8a8Srgb,
		vk.ImageTilingOptimal,
		vk.ImageUsageFlags(vk.ImageUsageTransferDstBit|vk.ImageUsageSampledBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit),
	)

	c.transitionImageLayout(c.textureImage, vk.FormatR8g8b8a8Srgb, vk.ImageLayoutUndefined, vk.ImageLayoutTransferDstOptimal)
	c.copyBufferToImage(stgBuf.Handle, c.textureImage, uint32(w), uint32(h))
	c.transitionImageLayout(c.textureImage, vk.FormatR8g8b8a8Srgb, vk.ImageLayoutTransferDstOptimal, vk.ImageLayoutShaderReadOnlyOptimal)

	vk.DestroyBuffer(c.device.D, stgBuf.Handle, nil)
	vk.FreeMemory(c.device.D, stgBuf.DeviceMem, nil)
}

func (c *Core) createTextureViews() {
	c.textureImageView = c.createImageView(c.textureImage, vk.FormatR8g8b8a8Srgb, vk.ImageAspectFlags(vk.ImageAspectColorBit))
}

func (c *Core) createTextureSampler() {
	samplerInfo := &vk.SamplerCreateInfo{
		SType:                   vk.StructureTypeSamplerCreateInfo,
		PNext:                   nil,
		Flags:                   0,
		MagFilter:               vk.FilterLinear,
		MinFilter:               vk.FilterLinear,
		MipmapMode:              vk.SamplerMipmapModeLinear,
		AddressModeU:            vk.SamplerAddressModeRepeat,
		AddressModeV:            vk.SamplerAddressModeRepeat,
		AddressModeW:            vk.SamplerAddressModeRepeat,
		MipLodBias:              0.0,
		AnisotropyEnable:        vk.True,
		MaxAnisotropy:           c.device.PdProps.Limits.MaxSamplerAnisotropy,
		CompareEnable:           vk.False,
		CompareOp:               vk.CompareOpAlways,
		MinLod:                  0.0,
		MaxLod:                  0.0,
		BorderColor:             vk.BorderColorIntOpaqueBlack,
		UnnormalizedCoordinates: vk.False,
	}
	var sampler vk.Sampler
	if vk.CreateSampler(c.device.D, samplerInfo, nil, &sampler) != vk.Success {
		log.Panicf("Failed to create texture sampler")
	}
	c.textureSampler = sampler
}

func (c *Core) createDepthResources() {
	dFormat := c.findDepthFormat()
	dImg, dImgMem := com.CreateImage(
		c.device,
		c.swapChain.Extend.Width,
		c.swapChain.Extend.Height,
		dFormat,
		vk.ImageTilingOptimal,
		vk.ImageUsageFlags(vk.ImageUsageDepthStencilAttachmentBit),
		vk.MemoryPropertyFlags(vk.MemoryPropertyDeviceLocalBit),
	)
	dImgView := c.createImageView(dImg, dFormat, vk.ImageAspectFlags(vk.ImageAspectDepthBit))
	c.depthImage = dImg
	c.depthImageMem = dImgMem
	c.depthImageView = dImgView

	c.transitionImageLayout(c.depthImage, dFormat, vk.ImageLayoutUndefined, vk.ImageLayoutDepthStencilAttachmentOptimal)
}

func (c *Core) findDepthFormat() vk.Format {
	return c.findSupportedFormat(
		[]vk.Format{vk.FormatD32Sfloat, vk.FormatD32SfloatS8Uint, vk.FormatD24UnormS8Uint},
		vk.ImageTilingOptimal,
		vk.FormatFeatureFlags(vk.FormatFeatureDepthStencilAttachmentBit),
	)
}

func hasStencilComponent(format vk.Format) bool {
	return format == vk.FormatD32SfloatS8Uint || format == vk.FormatD24UnormS8Uint
}

func (c *Core) findSupportedFormat(candidates []vk.Format, tiling vk.ImageTiling, features vk.FormatFeatureFlags) vk.Format {
	for _, format := range candidates {
		var fProps vk.FormatProperties
		vk.GetPhysicalDeviceFormatProperties(c.device.PD, format, &fProps)
		fProps.Deref()
		if tiling == vk.ImageTilingLinear && (fProps.LinearTilingFeatures&features) == features {
			return format
		} else if tiling == vk.ImageTilingOptimal && (fProps.OptimalTilingFeatures&features) == features {
			return format
		}
	}
	panic("No supported format found")
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
		Extent: c.swapChain.Extend,
	}
	clearValues := []vk.ClearValue{
		vk.NewClearValue([]float32{0.01, 0.01, 0.01, 1}), // color
		vk.NewClearDepthStencil(1, 0),                    // depthStencil <- Go bindings are strange here ! dont really know about the necessary values
	}
	renderPassInfo := vk.RenderPassBeginInfo{
		SType:           vk.StructureTypeRenderPassBeginInfo,
		PNext:           nil,
		RenderPass:      c.renderPass,
		Framebuffer:     c.swapChain.FrameBuffers[imageIdx],
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
			Width:    float32(c.swapChain.Extend.Width),
			Height:   float32(c.swapChain.Extend.Height),
			MinDepth: 0,
			MaxDepth: 1.0,
		},
	}
	vk.CmdSetViewport(buffer, 0, 1, viewport)

	scissor := []vk.Rect2D{
		{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: c.swapChain.Extend,
		},
	}
	vk.CmdSetScissor(buffer, 0, 1, scissor)
	vk.CmdBindDescriptorSets(buffer, vk.PipelineBindPointGraphics, c.pipelineLayout, 0, 1, []vk.DescriptorSet{c.descriptorSets[imageIdx]}, 0, nil)

	for i := range c.models {
		vertBuffers := []vk.Buffer{c.models[i].VertexBuffer}
		offsets := []vk.DeviceSize{0}
		vk.CmdBindVertexBuffers(buffer, 0, uint32(len(vertBuffers)), vertBuffers, offsets)
		vk.CmdBindIndexBuffer(buffer, c.models[i].IndexBuffer, 0, vk.IndexTypeUint32)
		pPConst := com.UnsafeMatPtr(&c.models[i].Mesh.ModelMat)
		vk.CmdPushConstants(buffer, c.pipelineLayout, vk.ShaderStageFlags(vk.ShaderStageVertexBit), 0, model.ModelPushConstantsSize(), pPConst)
		vk.CmdDrawIndexed(buffer, uint32(len(c.models[i].Mesh.VIndices)), 1, 0, 0, 0)
	}

	vk.CmdEndRenderPass(buffer)
	if vk.EndCommandBuffer(buffer) != vk.Success {
		log.Printf("Failed to record commandbuffer")
	}
}

func (c *Core) drawFrame() {
	// Wait for frame to be ready - signalled by the inFlightFens
	vk.WaitForFences(c.device.D, 1, []vk.Fence{c.inFlightFens[c.currentFrameIdx]}, vk.True, math.MaxUint64)

	var imgIdx uint32
	result := vk.AcquireNextImage(c.device.D, c.swapChain.Handle, math.MaxUint64, c.imageAvailableSems[c.currentFrameIdx], nil, &imgIdx)
	// React on surface changes and other possible causes for failure (e.g.: Window resizing)
	if result == vk.ErrorOutOfDate {
		c.recreateSwapChain()
		return
	} else if result != vk.Success && result != vk.Suboptimal {
		log.Panicf("Failed to aquire image, AcquireNextImage(...) result code: %d", result)
	}

	// Reset the fence only if we are actually going to execute work that will put the fence into the signalled state
	vk.ResetFences(c.device.D, 1, []vk.Fence{c.inFlightFens[c.currentFrameIdx]})

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
	if vk.QueueSubmit(c.device.GraphicsQ, 1, []vk.SubmitInfo{submitInfo}, c.inFlightFens[c.currentFrameIdx]) != vk.Success {
		log.Panicf("Failed to submit commandbuffer")
	}

	presentInfo := vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		PNext:              nil,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{c.renderFinishedSems[c.currentFrameIdx]},
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{c.swapChain.Handle},
		PImageIndices:      []uint32{imgIdx},
		PResults:           nil,
	}
	result = vk.QueuePresent(c.device.PresentQ, &presentInfo)
	// React on surface changes and other possible causes for failure (e.g.: Window resizing)
	if result == vk.ErrorOutOfDate || result == vk.Suboptimal || c.win.Resized {
		c.win.Resized = false
		c.recreateSwapChain()
	} else if result != vk.Success {
		log.Panicf("Failed to present image, QueuePresent(...) result code: %d", result)
	}

	c.currentFrameIdx = (c.currentFrameIdx + 1) % MAX_FRAMES_IN_FLIGHT
}

func (c *Core) recreateSwapChain() {
	vk.DeviceWaitIdle(c.device.D)
	c.destroySwapChainAndDerivatives()
	c.swapChain = com.NewSwapChain(c.device, c.win)
	c.createDepthResources()
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
	textureSamplerLayoutBinding := vk.DescriptorSetLayoutBinding{
		Binding:            1,                                     // <- binding index in frag shader
		DescriptorType:     vk.DescriptorTypeCombinedImageSampler, // <- type of binding in frag shader
		DescriptorCount:    1,
		StageFlags:         vk.ShaderStageFlags(vk.ShaderStageFragmentBit),
		PImmutableSamplers: nil,
	}
	layoutInfo := vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		PNext:        nil,
		Flags:        0,
		BindingCount: 2,
		PBindings:    []vk.DescriptorSetLayoutBinding{uboLayoutBinding, textureSamplerLayoutBinding},
	}
	var dsl vk.DescriptorSetLayout
	if vk.CreateDescriptorSetLayout(c.device.D, &layoutInfo, nil, &dsl) != vk.Success {
		log.Panicf("Failed to create descriptor set layout")
	}
	c.descriptorSetLayout = dsl
}

func (c *Core) createUniformBuffers() {
	uboBufSize := model.SizeOfUbo()
	log.Printf("UBO buffer size: %d Byte", uboBufSize)

	c.uniformBuffers = make([]vk.Buffer, MAX_FRAMES_IN_FLIGHT)
	c.uniformBufferMems = make([]vk.DeviceMemory, MAX_FRAMES_IN_FLIGHT)
	c.uniformBuffersMapped = make([]unsafe.Pointer, MAX_FRAMES_IN_FLIGHT)

	memProps := vk.MemoryPropertyFlags(vk.MemoryPropertyHostVisibleBit | vk.MemoryPropertyHostCoherentBit)
	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		uboBuf := com.CreateBuffer(
			c.device,
			uboBufSize,
			vk.BufferUsageFlags(vk.BufferUsageUniformBufferBit),
			memProps,
		)
		c.uniformBuffers[i] = uboBuf.Handle
		c.uniformBufferMems[i] = uboBuf.DeviceMem
		vk.MapMemory(c.device.D, c.uniformBufferMems[i], 0, uboBufSize, 0, &c.uniformBuffersMapped[i])
	}
}

func (c *Core) createDescriptorPool() {
	uboPoolSize := vk.DescriptorPoolSize{
		Type:            vk.DescriptorTypeUniformBuffer,
		DescriptorCount: MAX_FRAMES_IN_FLIGHT,
	}
	texSamplerPoolSize := vk.DescriptorPoolSize{
		Type:            vk.DescriptorTypeCombinedImageSampler,
		DescriptorCount: MAX_FRAMES_IN_FLIGHT,
	}
	poolInfo := vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		PNext:         nil,
		Flags:         0,
		MaxSets:       MAX_FRAMES_IN_FLIGHT,
		PoolSizeCount: 2,
		PPoolSizes:    []vk.DescriptorPoolSize{uboPoolSize, texSamplerPoolSize},
	}
	var dp vk.DescriptorPool
	if vk.CreateDescriptorPool(c.device.D, &poolInfo, nil, &dp) != vk.Success {
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
	if vk.AllocateDescriptorSets(c.device.D, &allocInfo, &(sets[0])) != vk.Success {
		log.Panicf("Failed to allocate descriptor set")
	}
	log.Printf("%v", sets)
	c.descriptorSets = sets

	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		// ubo
		bufferInfo := vk.DescriptorBufferInfo{
			Buffer: c.uniformBuffers[i],
			Offset: 0,
			Range:  model.SizeOfUbo(),
		}
		uboDescriptorWrite := vk.WriteDescriptorSet{
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

		// textureSampler
		texSampler := vk.DescriptorImageInfo{
			Sampler:     c.textureSampler,
			ImageView:   c.textureImageView,
			ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal,
		}
		texSamplerDescriptorWrite := vk.WriteDescriptorSet{
			SType:            vk.StructureTypeWriteDescriptorSet,
			PNext:            nil,
			DstSet:           c.descriptorSets[i],
			DstBinding:       1,
			DstArrayElement:  0,
			DescriptorCount:  1,
			DescriptorType:   vk.DescriptorTypeCombinedImageSampler,
			PImageInfo:       []vk.DescriptorImageInfo{texSampler},
			PBufferInfo:      nil,
			PTexelBufferView: nil,
		}
		writes := []vk.WriteDescriptorSet{uboDescriptorWrite, texSamplerDescriptorWrite}
		vk.UpdateDescriptorSets(c.device.D, uint32(len(writes)), writes, 0, nil)
	}
}

func (c *Core) updateUniformBuffer(frameIdx int32) {
	c.Cam.Aspect = c.swapChain.Aspect
	ubo := model.UniformBufferObject{
		View:       c.Cam.GetView(),
		Projection: c.Cam.GetProjection(),
	}
	vk.Memcopy(c.uniformBuffersMapped[frameIdx], ubo.Bytes())
}
