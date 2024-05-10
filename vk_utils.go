package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	vk "github.com/goki/vulkan"
	"log"
	"os"
	"unsafe"
)

// Read operations that require duplicated function calls, allocations and dereferencing
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

func readShaderCode(device vk.Device, shaderFile string) vk.ShaderModule {
	shaderCodeB, err := os.ReadFile(shaderFile)
	shaderCodeLen := uint64(len(shaderCodeB))
	if err != nil {
		log.Panicf("Failed to read shader file: '%s' due to: %v", shaderFile, err)
	}
	log.Printf("Read shader file (%s) of size: %dByte", shaderFile, shaderCodeLen)

	createInfo := &vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		PNext:    nil,
		Flags:    0,
		CodeSize: shaderCodeLen,
		PCode:    sliceUint32(shaderCodeB),
	}
	var shaderModule vk.ShaderModule
	if vk.CreateShaderModule(device, createInfo, nil, &shaderModule) != vk.Success {
		log.Panicf("Failed to create shader module: '%s'", shaderFile)
	}
	return shaderModule
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

// Support and availability checks
func checkInstanceExtensionSupport(requiredInstanceExt []string) {
	supportedExt := readInstanceExtensionProperties()
	log.Printf("Required instance extensions: %v", requiredInstanceExt)
	log.Printf("Available extensions (%d):\n%v", len(supportedExt), tableStringExtensionProps(supportedExt))
	supportedExtNames := make([]string, len(supportedExt))
	for i, ext := range supportedExt {
		supportedExtNames[i] = vk.ToString(ext.ExtensionName[:])
	}

	if !allOfAinB(requiredInstanceExt, supportedExtNames) {
		log.Panicf("At least one required instance extension is not supported")
	} else {
		log.Println("Success - All required instance extensions are supported")
	}
}

func checkValidationLayerSupport(requiredLayers []string) {
	supportedLayers := readInstanceLayerProperties()
	log.Printf("Desired validation layers: %v", requiredLayers)
	log.Printf("Supported layers (%d):\n%v", len(supportedLayers), tableStringLayerProps(supportedLayers))

	supLayerNames := make([]string, len(supportedLayers))
	for i, l := range supportedLayers {
		supLayerNames[i] = vk.ToString(l.LayerName[:])
	}

	if !allOfAinB(requiredLayers, supLayerNames) {
		log.Panicf("At least one desired layers are supported")
	} else {
		log.Println("Success - All desired validation layers are supported")
	}
}

func checkDeviceExtensionSupport(pd vk.PhysicalDevice, requiredDeviceExt []string) bool {
	supportedExt := readDeviceExtensionProperties(pd)
	log.Printf("Required device extensions: %v", requiredDeviceExt)
	log.Printf("Available device extensions (%d) [...]\n", len(supportedExt))
	//log.Printf("Available device extensions (%d):\n%v", len(supportedExt), tableStringExtensionProps(supportedExt))
	supportedExtNames := make([]string, len(supportedExt))
	for i, ext := range supportedExt {
		supportedExtNames[i] = vk.ToString(ext.ExtensionName[:])
	}
	return allOfAinB(requiredDeviceExt, supportedExtNames)
}

func checkSwapChainAdequacy(pd vk.PhysicalDevice, surface vk.Surface) bool {
	scDetails := readSwapChainSupportDetails(pd, surface)
	log.Printf("Read swap chain details: %v", scDetails)
	return len(scDetails.formats) > 0 && len(scDetails.presentModes) > 0
}

// Comparisons and misc tooling
func allOfAinB(a []string, b []string) bool {
	for _, _a := range a {
		isIn := false
		for _, _b := range b {
			if _a == _b {
				isIn = true
				break
			}
		}
		if !isIn {
			return false
		}
	}
	return true
}

func rawBytes(p interface{}) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, p)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

func terminatedStr(s string) string {
	if s[len(s)-1] != '\x00' {
		return s + "\x00"
	}
	return s
}

func terminatedStrs(strs []string) []string {
	for i := range strs {
		strs[i] = terminatedStr(strs[i])
	}
	return strs
}

// Nasty conversion logic taken from: https://github.com/vulkan-go/asche/blob/master/util.go
// it should be equivalent to C++ 'reinterpret_cast<const uint32_t*>(code.data());'
// See: https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Shader_modules
func sliceUint32(data []byte) []uint32 {
	const m = 0x7fffffff
	return (*[m / 4]uint32)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&data)).Data))[:len(data)/4]
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
