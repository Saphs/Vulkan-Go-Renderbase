package common

import (
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

// Device encapsulates all device information, bundling vk.PhysicalDevice, vk.Device and their auxiliary information
// into a neat package. It is meant as the primary way to access the vk.Device information by core rendering components.
// Thus acting as the first layer of abstraction over raw Vulkan API calls.
type Device struct {
	PD            vk.PhysicalDevice
	PdProps       vk.PhysicalDeviceProperties
	PdMemoryProps vk.PhysicalDeviceMemoryProperties
	QFamilies     QueueFamilyIndices

	D         vk.Device
	GraphicsQ vk.Queue
	PresentQ  vk.Queue
}

// NewDevice constructs a new Device struct as described above. This includes device selection and is therefore,
// application specific. Requirements for the GPU we want to work with, are currently (23/03/25) a minimal set imposing
// only a few common constraint like: Having a graphics and present queue, allowing for anisotropic filtering and
// supporting basic validation layers.
func NewDevice(w *Window) *Device {
	dc := &Device{}
	dc.selectPhysicalDevice(w.Inst, w.Surf)
	dc.createLogicalDevice()
	return dc
}

// Destroy is a convenience function wrapping the vk.DestroyDevice used to destroy the logical device which is the
// actual resource we need to destroy on teardown.
func (dc *Device) Destroy() {
	vk.DestroyDevice(dc.D, nil)
}

// ToDo: Reading out the physical device properties (multiple times) is very clunky here. This could/should be
// refactored once device selection becomes more stringent.
func (dc *Device) selectPhysicalDevice(in *vk.Instance, su *vk.Surface) {
	availableDevices := ReadPhysicalDevices(*in)
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
	log.Printf("Found suitable device: \"%s\"")
	dc.PD = pd

	// Also set related member variables for dc.physicalDevice as they are needed later
	qf, err := findQueueFamilies(dc.PD, *su)
	if err != nil {
		log.Panicf("Failed to read queue families from selected device due to: %s", err)
	}
	dc.QFamilies = *qf
	dc.PdProps = ReadPhysicalDeviceProperties(dc.PD)
	// this is the easiest spot to deref this at the moment
	dc.PdProps.Limits.Deref()
	dc.PdMemoryProps = ReadDeviceMemoryProperties(dc.PD)
}

func isDeviceSuitable(pd vk.PhysicalDevice, su *vk.Surface) bool {
	pdProps := ReadPhysicalDeviceProperties(pd)
	pdFeatures := ReadPhysicalDeviceFeatures(pd)
	pdQueueFams := ReadQueueFamilies(pd)

	log.Printf("Physical divece\n%s", ToStringPhysicalDeviceTable(pdProps, pdFeatures, pdQueueFams))

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

func (dc *Device) createLogicalDevice() {
	queueInfos := dc.QFamilies.toQueueCreateInfos()
	// We explicitly enable anisotropic sampling, more interesting stuff could be added here
	deviceFeatures := vk.PhysicalDeviceFeatures{
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
		PpEnabledExtensionNames: TerminatedStrs(DEVICE_EXTENSIONS),
		PEnabledFeatures:        []vk.PhysicalDeviceFeatures{deviceFeatures},
	}
	if ENABLE_VALIDATION {
		deviceCreatInfo.EnabledLayerCount = uint32(len(VALIDATION_LAYERS))
		deviceCreatInfo.PpEnabledLayerNames = TerminatedStrs(VALIDATION_LAYERS)
	}

	var err error
	dc.D, err = VkCreateDevice(dc.PD, deviceCreatInfo, nil)
	if err != nil {
		log.Panicf("Failed create logical device due to: %s", "err")
	}
	dc.GraphicsQ, err = VkGetDeviceQueue(dc.D, dc.QFamilies.GraphicsFamily, 0)
	if err != nil {
		log.Panicf("Failed to get 'graphics' device queue: %s", err)
	}
	dc.PresentQ, err = VkGetDeviceQueue(dc.D, dc.QFamilies.PresentFamily, 0)
	if err != nil {
		log.Panicf("Failed to get 'present' device queue: %s", err)
	}
}

func checkDeviceExtensionSupport(pd vk.PhysicalDevice, requiredDeviceExt []string) bool {
	supportedExt := ReadDeviceExtensionProperties(pd)
	log.Printf("Required device extensions: %v", requiredDeviceExt)
	log.Printf("Available device extensions (%d) [...]\n", len(supportedExt))
	//log.Printf("Available device extensions (%d):\n%v", len(supportedExt), tableStringExtensionProps(supportedExt))
	supportedExtNames := make([]string, len(supportedExt))
	for i, ext := range supportedExt {
		supportedExtNames[i] = vk.ToString(ext.ExtensionName[:])
	}
	return IsSubset(requiredDeviceExt, supportedExtNames)
}
