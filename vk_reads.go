package main

import (
	vk "github.com/goki/vulkan"
	"log"
)

// Read operations that require duplicated function calls, allocations and dereferencing. It is pulled out to
// provide a more go-lang feel and tidy the core code.

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

func readPhysicalDevices(instance vk.Instance) []vk.PhysicalDevice {
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

func readPhysicalDeviceProperties(pd vk.PhysicalDevice) vk.PhysicalDeviceProperties {
	var pdProps vk.PhysicalDeviceProperties
	vk.GetPhysicalDeviceProperties(pd, &pdProps)
	pdProps.Deref()
	return pdProps
}

func readPhysicalDeviceFeatures(pd vk.PhysicalDevice) vk.PhysicalDeviceFeatures {
	var pdFeatures vk.PhysicalDeviceFeatures
	vk.GetPhysicalDeviceFeatures(pd, &pdFeatures)
	pdFeatures.Deref()
	return pdFeatures
}

func readQueueFamilies(pd vk.PhysicalDevice) []vk.QueueFamilyProperties {
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

func readDeviceExtensionProperties(pd vk.PhysicalDevice) []vk.ExtensionProperties {
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

func readSwapChainSupportDetails(pd vk.PhysicalDevice, surface vk.Surface) SwapChainDetails {
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

func readSwapChainImages(device vk.Device, swapChain vk.Swapchain) []vk.Image {
	var imgCount uint32
	vk.GetSwapchainImages(device, swapChain, &imgCount, nil)
	imgs := make([]vk.Image, imgCount)
	vk.GetSwapchainImages(device, swapChain, &imgCount, imgs)
	return imgs
}

func readDeviceMemoryProperties(pd vk.PhysicalDevice) vk.PhysicalDeviceMemoryProperties {
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

func readBufferMemoryRequirements(device vk.Device, b vk.Buffer) vk.MemoryRequirements {
	var memRequirements vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(device, b, &memRequirements)
	memRequirements.Deref()
	return memRequirements
}

func readImageMemoryRequirements(device vk.Device, img vk.Image) vk.MemoryRequirements {
	var memRequirements vk.MemoryRequirements
	vk.GetImageMemoryRequirements(device, img, &memRequirements)
	memRequirements.Deref()
	return memRequirements
}
