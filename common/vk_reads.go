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
