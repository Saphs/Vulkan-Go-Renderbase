package main

import (
	vk "github.com/goki/vulkan"
	"github.com/veandco/go-sdl2/sdl"
	"log"
)

// DeviceContext represents the interfacing objects between the SDL window, the Hardware running Vulkan
// and the rest of the rendering engine. Its main purpose is to encapsulate the corresponding objects
// to make the initialization and teardown of a given application neater.
type DeviceContext struct {
	win        *sdl.Window
	vkInstance vk.Instance
	vkSurface  vk.Surface

	physicalDevice vk.PhysicalDevice
	pdMemoryProps  vk.PhysicalDeviceMemoryProperties
	qFamilies      QueueFamilyIndices

	device    vk.Device
	graphicsQ vk.Queue
	presentQ  vk.Queue
}

func NewDeviceContext(w *sdl.Window) *DeviceContext {
	return &DeviceContext{
		win: w,
	}
}

func (dc *DeviceContext) create() {
	dc.createInstance()
	dc.createSurface()
	dc.selectPhysicalDevice()
	dc.createLogicalDevice()
}

// destroy all objects created by itself. It does not destroy the sdl.window object provided for instantiation.
func (dc *DeviceContext) destroy() {
	vk.DestroySurface(dc.vkInstance, dc.vkSurface, nil)
	vk.DestroyDevice(dc.device, nil)
	vk.DestroyInstance(dc.vkInstance, nil)
}

func (dc *DeviceContext) createInstance() {
	requiredExtensions := dc.win.VulkanGetInstanceExtensions()
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
	dc.vkInstance = ins
}

func (dc *DeviceContext) createSurface() {
	surf, err := sdlCreateVkSurface(dc.win, dc.vkInstance)
	if err != nil {
		log.Panicf("Failed to create SDL window's Vulkan-surface, due to: %v", err)
	}
	dc.vkSurface = surf
}

func (dc *DeviceContext) selectPhysicalDevice() {
	availableDevices := readPhysicalDevices(dc.vkInstance)
	var pd vk.PhysicalDevice
	for i := range availableDevices {
		if isDeviceSuitable(availableDevices[i], dc.vkSurface) {
			pd = availableDevices[i]
			break
		}
	}
	if pd == nil {
		log.Panicf("No suitable physical device (GPU) found")
	}
	log.Printf("Found suitable device")
	dc.physicalDevice = pd

	// Also set related member variables for dc.physicalDevice as they are needed later
	qf, err := findQueueFamilies(dc.physicalDevice, dc.vkSurface)
	if err != nil {
		log.Panicf("Failed to read queue families from selected device due to: %s", err)
	}
	dc.qFamilies = *qf
	dc.pdMemoryProps = readDeviceMemoryProperties(dc.physicalDevice)
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

func (dc *DeviceContext) createLogicalDevice() {
	queueInfos := dc.qFamilies.toQueueCreateInfos()
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
	dc.device, err = VkCreateDevice(dc.physicalDevice, deviceCreatInfo, nil)
	if err != nil {
		log.Panicf("Failed create logical device due to: %s", "err")
	}
	dc.graphicsQ, err = VkGetDeviceQueue(dc.device, dc.qFamilies.graphicsFamily, 0)
	if err != nil {
		log.Panicf("Failed to get 'graphics' device queue: %s", err)
	}
	dc.presentQ, err = VkGetDeviceQueue(dc.device, dc.qFamilies.presentFamily, 0)
	if err != nil {
		log.Panicf("Failed to get 'present' device queue: %s", err)
	}
}
