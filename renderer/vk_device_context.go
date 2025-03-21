package renderer

import (
	"GPU_fluid_simulation/common"
	vk "github.com/goki/vulkan"
	"log"
)

const ENABLE_VALIDATION = true

var VALIDATION_LAYERS = []string{
	"VK_LAYER_KHRONOS_validation",
}

var DEVICE_EXTENSIONS = []string{
	"VK_KHR_swapchain",
}

// DeviceContext represents the interfacing objects between the SDL window, the Hardware running Vulkan
// and the rest of the rendering engine. Its main purpose is to encapsulate the corresponding objects
// to make the initialization and teardown of a given application neater.
type DeviceContext struct {
	physicalDevice vk.PhysicalDevice
	pdProps        vk.PhysicalDeviceProperties
	pdMemoryProps  vk.PhysicalDeviceMemoryProperties
	qFamilies      QueueFamilyIndices

	device    vk.Device
	graphicsQ vk.Queue
	presentQ  vk.Queue
}

func NewDeviceContext(w *common.Window) *DeviceContext {
	dc := &DeviceContext{}
	dc.selectPhysicalDevice(w.Inst, w.Surf)
	dc.createLogicalDevice()
	return dc
}

// destroy all objects created by itself. It does not destroy the sdl.window object provided for instantiation.
func (dc *DeviceContext) destroy() {
	vk.DestroyDevice(dc.device, nil)
}

func (dc *DeviceContext) selectPhysicalDevice(in *vk.Instance, su *vk.Surface) {
	availableDevices := readPhysicalDevices(*in)
	var pd vk.PhysicalDevice
	for i := range availableDevices {
		if isDeviceSuitable(availableDevices[i], su) {
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
	qf, err := findQueueFamilies(dc.physicalDevice, *su)
	if err != nil {
		log.Panicf("Failed to read queue families from selected device due to: %s", err)
	}
	dc.qFamilies = *qf
	dc.pdProps = readPhysicalDeviceProperties(dc.physicalDevice)
	// this is the easiest spot to deref this at the moment
	dc.pdProps.Limits.Deref()
	dc.pdMemoryProps = readDeviceMemoryProperties(dc.physicalDevice)
}

func isDeviceSuitable(pd vk.PhysicalDevice, su *vk.Surface) bool {
	pdProps := readPhysicalDeviceProperties(pd)
	pdFeatures := readPhysicalDeviceFeatures(pd)
	pdQueueFams := readQueueFamilies(pd)

	log.Printf("Physical divece\n%s", common.ToStringPhysicalDeviceTable(pdProps, pdFeatures, pdQueueFams))

	indices, err := findQueueFamilies(pd, *su)
	if err != nil {
		log.Printf("Failed to get required queue families: %s", err)
		return false
	}

	queuesSupported := indices.isAllQueuesFound()
	isDiscreteGPU := pdProps.DeviceType == vk.PhysicalDeviceTypeDiscreteGpu
	featuresSupported := pdFeatures.GeometryShader == vk.True && pdFeatures.SamplerAnisotropy == vk.True
	extensionsSupported := checkDeviceExtensionSupport(pd, DEVICE_EXTENSIONS)

	isSwapChainAdequate := false
	if extensionsSupported {
		isSwapChainAdequate = checkSwapChainAdequacy(pd, *su)
	}

	return isDiscreteGPU && featuresSupported && queuesSupported && extensionsSupported && isSwapChainAdequate
}

func (dc *DeviceContext) createLogicalDevice() {
	queueInfos := dc.qFamilies.toQueueCreateInfos()
	deviceFeatures := vk.PhysicalDeviceFeatures{ // We explicitly enable anisotropic sampling, more interesting stuff could be added here
		SamplerAnisotropy: vk.True,
	}
	deviceCreatInfo := &vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		PNext:                   nil,
		Flags:                   0,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PQueueCreateInfos:       queueInfos,
		EnabledLayerCount:       0,
		PpEnabledLayerNames:     nil,
		EnabledExtensionCount:   uint32(len(DEVICE_EXTENSIONS)),
		PpEnabledExtensionNames: common.TerminatedStrs(DEVICE_EXTENSIONS),
		PEnabledFeatures:        []vk.PhysicalDeviceFeatures{deviceFeatures},
	}
	if ENABLE_VALIDATION {
		deviceCreatInfo.EnabledLayerCount = uint32(len(VALIDATION_LAYERS))
		deviceCreatInfo.PpEnabledLayerNames = common.TerminatedStrs(VALIDATION_LAYERS)
	}

	var err error
	dc.device, err = common.VkCreateDevice(dc.physicalDevice, deviceCreatInfo, nil)
	if err != nil {
		log.Panicf("Failed create logical device due to: %s", "err")
	}
	dc.graphicsQ, err = common.VkGetDeviceQueue(dc.device, dc.qFamilies.graphicsFamily, 0)
	if err != nil {
		log.Panicf("Failed to get 'graphics' device queue: %s", err)
	}
	dc.presentQ, err = common.VkGetDeviceQueue(dc.device, dc.qFamilies.presentFamily, 0)
	if err != nil {
		log.Panicf("Failed to get 'present' device queue: %s", err)
	}
}
