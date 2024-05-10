package main

import (
	vk "github.com/goki/vulkan"
	"log"
)

// Provides validation functions to ensure support and availability of requirements of layers/extensions and so on.

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
