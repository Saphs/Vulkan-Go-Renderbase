package common

import (
	vk "github.com/goki/vulkan"
	"log"
)

// ReadInstanceExtensionPropertyNames is a convenience method obfuscating the spec defined []vk.ExtensionProperties
// type in favor of their respective names in order to simplify support checks to a point of string comparisons.
func ReadInstanceExtensionPropertyNames() []string {
	supportedExts := readInstanceExtensionProperties()
	supportedExtNames := make([]string, len(supportedExts))
	for i, ext := range supportedExts {
		supportedExtNames[i] = vk.ToString(ext.ExtensionName[:])
	}
	return supportedExtNames
}

// readInstanceExtensionProperties wraps the raw vulkan call to retrieve all supported instance extensions as their
// spec defined type and dereferences all necessary pointer values.
func readInstanceExtensionProperties() []vk.ExtensionProperties {
	extensionCount := uint32(0)
	err := vk.Error(vk.EnumerateInstanceExtensionProperties("", &extensionCount, nil))
	if err != nil {
		log.Panicf("Failed read number of InstanceExtensionProperties: %s", err)
	}
	extensionProperties := make([]vk.ExtensionProperties, extensionCount)
	err = vk.Error(vk.EnumerateInstanceExtensionProperties("", &extensionCount, extensionProperties))
	if err != nil {
		log.Panicf("Failed read %d InstanceExtensionProperties: %s", extensionCount, err)
	}
	for i := range extensionProperties {
		extensionProperties[i].Deref()
	}
	return extensionProperties
}

// ReadInstanceLayerPropertyNames is a convenience method obfuscating the spec defined []vk.LayerProperties
// type in favor of their respective names in order to simplify support checks to a point of string comparisons.
func ReadInstanceLayerPropertyNames() []string {
	supportedLayers := readInstanceLayerProperties()
	supLayerNames := make([]string, len(supportedLayers))
	for i, l := range supportedLayers {
		supLayerNames[i] = vk.ToString(l.LayerName[:])
	}
	return supLayerNames
}

// readInstanceLayerProperties wraps the raw vulkan call to retrieve all supported instance (validation) layer
// properties as their spec defined type and dereferences all necessary pointer values.
func readInstanceLayerProperties() []vk.LayerProperties {
	layerCount := uint32(0)
	err := vk.Error(vk.EnumerateInstanceLayerProperties(&layerCount, nil))
	if err != nil {
		log.Panicf("Failed read number of InstanceLayerProperties: %s", err)
	}
	layers := make([]vk.LayerProperties, layerCount)
	err = vk.Error(vk.EnumerateInstanceLayerProperties(&layerCount, layers))
	if err != nil {
		log.Panicf("Failed read %d InstanceLayerProperties: %s", layerCount, err)
	}
	for i := range layers {
		layers[i].Deref()
	}
	return layers
}

func ReadSwapChainSupportDetails(pd vk.PhysicalDevice, surface vk.Surface) SwapChainDetails {
	scDetails := SwapChainDetails{}
	vk.GetPhysicalDeviceSurfaceCapabilities(pd, surface, &scDetails.capabilities)
	scDetails.capabilities.Deref()

	var formatCount uint32
	vk.GetPhysicalDeviceSurfaceFormats(pd, surface, &formatCount, nil)
	scDetails.formats = make([]vk.SurfaceFormat, formatCount)
	vk.GetPhysicalDeviceSurfaceFormats(pd, surface, &formatCount, scDetails.formats)
	for i := range scDetails.formats {
		scDetails.formats[i].Deref()
	}

	var presentModeCount uint32
	vk.GetPhysicalDeviceSurfacePresentModes(pd, surface, &presentModeCount, nil)
	scDetails.presentModes = make([]vk.PresentMode, presentModeCount)
	vk.GetPhysicalDeviceSurfacePresentModes(pd, surface, &presentModeCount, scDetails.presentModes)

	return scDetails
}

func ReadSwapChainImages(device vk.Device, swapChain vk.Swapchain) []vk.Image {
	var imgCount uint32
	vk.GetSwapchainImages(device, swapChain, &imgCount, nil)
	imgs := make([]vk.Image, imgCount)
	vk.GetSwapchainImages(device, swapChain, &imgCount, imgs)
	return imgs
}

func ReadPhysicalDevices(instance vk.Instance) []vk.PhysicalDevice {
	var gpuCount uint32
	err := vk.Error(vk.EnumeratePhysicalDevices(instance, &gpuCount, nil))
	if err != nil {
		log.Panicf("Failed to read number of PhysicalDevices failed with: %s", err)
	}
	if gpuCount == 0 {
		log.Panic("There are 0 physical devices available")
	}
	physDevices := make([]vk.PhysicalDevice, gpuCount)
	err = vk.Error(vk.EnumeratePhysicalDevices(instance, &gpuCount, physDevices))
	if err != nil {
		log.Panicf("Failed to read %d PhysicalDevices failed with: %s", gpuCount, err)
	}
	return physDevices
}

func ReadDeviceMemoryProperties(pd vk.PhysicalDevice) vk.PhysicalDeviceMemoryProperties {
	var pdMemProps vk.PhysicalDeviceMemoryProperties
	vk.GetPhysicalDeviceMemoryProperties(pd, &pdMemProps)
	pdMemProps.Deref()
	for i := range pdMemProps.MemoryTypes {
		pdMemProps.MemoryTypes[i].Deref()
	}
	for i := range pdMemProps.MemoryHeaps {
		pdMemProps.MemoryHeaps[i].Deref()
	}
	return pdMemProps
}

func ReadPhysicalDeviceProperties(pd vk.PhysicalDevice) vk.PhysicalDeviceProperties {
	var pdProps vk.PhysicalDeviceProperties
	vk.GetPhysicalDeviceProperties(pd, &pdProps)
	pdProps.Deref()
	return pdProps
}

func ReadPhysicalDeviceFeatures(pd vk.PhysicalDevice) vk.PhysicalDeviceFeatures {
	var pdFeatures vk.PhysicalDeviceFeatures
	vk.GetPhysicalDeviceFeatures(pd, &pdFeatures)
	pdFeatures.Deref()
	return pdFeatures
}

func ReadQueueFamilies(pd vk.PhysicalDevice) []vk.QueueFamilyProperties {
	qFamilyCount := uint32(0)
	vk.GetPhysicalDeviceQueueFamilyProperties(pd, &qFamilyCount, nil)
	qFamilyProps := make([]vk.QueueFamilyProperties, qFamilyCount)
	vk.GetPhysicalDeviceQueueFamilyProperties(pd, &qFamilyCount, qFamilyProps)
	for i := range qFamilyProps {
		qFamilyProps[i].Deref()
		qFamilyProps[i].MinImageTransferGranularity.Deref()
	}
	return qFamilyProps
}

func ReadDeviceExtensionProperties(pd vk.PhysicalDevice) []vk.ExtensionProperties {
	extensionCount := uint32(0)
	err := vk.Error(vk.EnumerateDeviceExtensionProperties(pd, "", &extensionCount, nil))
	if err != nil {
		log.Panicf("Failed read number of DeviceExtensionProperties: %s", err)
	}
	extensionProperties := make([]vk.ExtensionProperties, extensionCount)
	err = vk.Error(vk.EnumerateDeviceExtensionProperties(pd, "", &extensionCount, extensionProperties))
	if err != nil {
		log.Panicf("Failed read %d DeviceExtensionProperties: %s", extensionCount, err)
	}
	for i := range extensionProperties {
		extensionProperties[i].Deref()
	}
	return extensionProperties
}
