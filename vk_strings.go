package main

import (
	"encoding/hex"
	"fmt"
	vk "github.com/goki/vulkan"
	"strings"
)

// ExtensionProperties
func tableStringExtensionProps(ext []vk.ExtensionProperties) string {
	strBuilder := strings.Builder{}
	for i := range ext {
		strBuilder.WriteString(fmt.Sprintf(" %s\n", toStringExtensionPropsTable(ext[i])))
	}
	return strBuilder.String()
}
func toStringExtensionPropsTable(e vk.ExtensionProperties) string {
	return fmt.Sprintf("%-59s%10s", vk.ToString(e.ExtensionName[:]), vk.Version(e.SpecVersion).String())
}
func toStringExtensionProps(e vk.ExtensionProperties) string {
	return fmt.Sprintf("%s, %s", vk.ToString(e.ExtensionName[:]), vk.Version(e.SpecVersion).String())
}

// LayerProperties
func tableStringLayerProps(lay []vk.LayerProperties) string {
	strBuilder := strings.Builder{}
	for i := range lay {
		strBuilder.WriteString(fmt.Sprintf(" %s\n", toStringLayerPropsTable(lay[i])))
	}
	return strBuilder.String()
}

func toStringLayerPropsTable(l vk.LayerProperties) string {
	return fmt.Sprintf(
		"%-40sspec: %8s   impl: %8s%50s",
		vk.ToString(l.LayerName[:]),
		vk.Version(l.SpecVersion).String(),
		vk.Version(l.ImplementationVersion).String(),
		vk.ToString(l.Description[:]),
	)
}

// Physical device
func toStringPhysicalDeviceTable(
	pdProps vk.PhysicalDeviceProperties,
	pdFeatures vk.PhysicalDeviceFeatures,
	qFamilies []vk.QueueFamilyProperties,
) string {
	strBuilder := strings.Builder{}
	for i := range qFamilies {
		if i == len(qFamilies)-1 {
			strBuilder.WriteString(fmt.Sprintf("|_Qfamily[%d] %s\n", i, toStringQueueFamilyPropsTable(qFamilies[i])))
		} else {
			strBuilder.WriteString(fmt.Sprintf("| Qfamily[%d] %s\n", i, toStringQueueFamilyPropsTable(qFamilies[i])))
		}
	}
	return fmt.Sprintf(
		"%s:\n|_%s\n|_%s\n%s",
		vk.ToString(pdProps.DeviceName[:]),
		toStringPhysicalDevicePropsTable(pdProps),
		toStringPhysicalDeviceFeatures(pdFeatures),
		strBuilder.String(),
	)
}

func asVendorName(v vk.VendorId) string {
	// There seem to only be a handful of vendors and Ids as stated in:
	// https://www.reddit.com/r/vulkan/comments/4ta9nj/is_there_a_comprehensive_list_of_the_names_and/
	switch v {
	case 0x1002:
		return "AMD"
	case 0x1010:
		return "ImgTec"
	case 0x10DE:
		return "NVIDIA"
	case 0x13B5:
		return "ARM"
	case 0x5143:
		return "Qualcomm"
	case 0x8086:
		return "INTEL"
	case 0x10005:
		return "Mesa"
	default:
		return "unknown"
	}
}

func asDriverVersion(vendor vk.VendorId, raw uint32) string {
	// Only nvidia and intel on windows are special.
	if vendor == 0x10DE { // NVIDIA
		return nvidiaVer(raw)
	} else {
		return vk.Version(raw).String()
	}
}

func nvidiaVer(i uint32) string {
	return fmt.Sprintf(
		"%d.%d.%d.%d",
		(i>>22)&0x3ff,
		(i>>14)&0x0ff,
		(i>>6)&0x0ff,
		i&0x003f,
	)
}

func toStringPhysicalDevicePropsTable(pdProps vk.PhysicalDeviceProperties) string {
	return fmt.Sprintf("api: %s, driver: %s, vendorId: %d (%s), deviceId: %d, deviceType: %d (%s), UUID: %v",
		vk.Version(pdProps.ApiVersion).String(),
		asDriverVersion(vk.VendorId(pdProps.VendorID), pdProps.DriverVersion),
		vk.VendorId(pdProps.VendorID),
		asVendorName(vk.VendorId(pdProps.VendorID)),
		pdProps.DeviceID,
		pdProps.DeviceType,
		toStringDeviceType(pdProps.DeviceType),
		hex.EncodeToString(pdProps.PipelineCacheUUID[:]),
	)
}

func toStringPhysicalDeviceProps(pdProps vk.PhysicalDeviceProperties) string {
	return fmt.Sprintf("PDevice(\"%s\", api: %s, driver: %s, vendorId: %d (%s), deviceId: %d, deviceType: %d (%s), UUID: %v)",
		vk.ToString(pdProps.DeviceName[:]),
		vk.Version(pdProps.ApiVersion).String(),
		asDriverVersion(vk.VendorId(pdProps.VendorID), pdProps.DriverVersion),
		vk.VendorId(pdProps.VendorID),
		asVendorName(vk.VendorId(pdProps.VendorID)),
		pdProps.DeviceID,
		pdProps.DeviceType,
		toStringDeviceType(pdProps.DeviceType),
		hex.EncodeToString(pdProps.PipelineCacheUUID[:]),
	)
}

func toStringPhysicalDeviceFeaturesTable(pdFeatures vk.PhysicalDeviceFeatures) string {
	return fmt.Sprintf("%v", pdFeatures)
}

func toStringPhysicalDeviceFeatures(pdFeatures vk.PhysicalDeviceFeatures) string {
	return fmt.Sprintf("PFeatures(%v)", pdFeatures)
}

func toStringDeviceType(dt vk.PhysicalDeviceType) string {
	switch dt {
	case 0:
		return "other"
	case 1:
		return "integrated Gpu"
	case 2:
		return "discrete Gpu"
	case 3:
		return "virtual Gpu"
	case 4:
		return "cpu"
	default:
		return "unknown"
	}
}

func toStringPhysicalDeviceMemProps(pdMemProps vk.PhysicalDeviceMemoryProperties) string {
	mtBuilder := strings.Builder{}
	mtBuilder.WriteString("\n")
	for i := uint32(0); i < pdMemProps.MemoryTypeCount; i++ {
		mt := pdMemProps.MemoryTypes[i]
		mtBuilder.WriteString(fmt.Sprintf(" %d: %s\n", i, toStringMemoryType(mt)))
	}
	mhBuilder := strings.Builder{}
	mhBuilder.WriteString("\n")
	for i := uint32(0); i < pdMemProps.MemoryHeapCount; i++ {
		mh := pdMemProps.MemoryHeaps[i]
		mhBuilder.WriteString(fmt.Sprintf(" %d: %s\n", i, toStringMemoryHeap(mh)))
	}
	return fmt.Sprintf(
		"PhysicalDeviceMemoryProperties(MemoryTypeCount: %d, MemoryTypes[%v], MemoryHeapCount: %d, MemoryHeaps[%v])",
		pdMemProps.MemoryTypeCount,
		mtBuilder.String(),
		pdMemProps.MemoryHeapCount,
		mhBuilder.String(),
	)
}

func toStringMemoryType(mt vk.MemoryType) string {
	return fmt.Sprintf("MemoryType(Flags:%032b, HeapIdx:%d)", mt.PropertyFlags, mt.HeapIndex)
}

func toStringMemoryHeap(mh vk.MemoryHeap) string {
	return fmt.Sprintf("MemoryHeap(Size:%d, Flags:%d)", mh.Size, mh.Flags)
}

func toStringMemoryRequirements(mr vk.MemoryRequirements) string {
	return fmt.Sprintf("MemoryRequirements(Size:%d Byte, Alignment:%d Byte, MemTypeBits:[%032b])", mr.Size, mr.Alignment, mr.MemoryTypeBits)
}

// QueueFamilyProperties
func tableStringQueueFamilyProps(qFamilies []vk.QueueFamilyProperties) string {
	builder := strings.Builder{}
	for i := range qFamilies {
		builder.WriteString(fmt.Sprintf("Q[%2d] %s\n", i, toStringQueueFamilyPropsTable(qFamilies[i])))
	}
	return builder.String()
}
func toStringQueueFamilyPropsTable(q vk.QueueFamilyProperties) string {
	return fmt.Sprintf(
		"Count: %2d, Valid ts bits: %d, ImageGranularity: (%d,%d,%d), Flags: %v",
		q.QueueCount,
		q.TimestampValidBits,
		q.MinImageTransferGranularity.Width,
		q.MinImageTransferGranularity.Height,
		q.MinImageTransferGranularity.Depth,
		toStringQueueFlags(q.QueueFlags),
	)
}

func toStringQueueFamilyProps(q vk.QueueFamilyProperties) string {
	return fmt.Sprintf(
		"QueueFamily(count: %2d, valid ts bits: %d, imageGranularity: (%d,%d,%d), flags: %v)",
		q.QueueCount,
		q.TimestampValidBits,
		q.MinImageTransferGranularity.Width,
		q.MinImageTransferGranularity.Height,
		q.MinImageTransferGranularity.Depth,
		toStringQueueFlags(q.QueueFlags),
	)
}

// QueueFlags
func toStringQueueFlags(bits vk.QueueFlags) []string {
	var properties []string
	flags := vk.QueueFlagBits(bits)
	if flags&vk.QueueGraphicsBit > 0 {
		properties = append(properties, "VK_QUEUE_GRAPHICS_BIT")
	}
	if flags&vk.QueueComputeBit > 0 {
		properties = append(properties, "VK_QUEUE_COMPUTE_BIT")
	}
	if flags&vk.QueueTransferBit > 0 {
		properties = append(properties, "VK_QUEUE_TRANSFER_BIT")
	}
	if flags&vk.QueueSparseBindingBit > 0 {
		properties = append(properties, "VK_QUEUE_SPARSE_BINDING_BIT")
	}
	if flags&vk.QueueProtectedBit > 0 {
		properties = append(properties, "VK_QUEUE_PROTECTED_BIT")
	}
	if flags&vk.QueueVideoDecodeBit > 0 {
		properties = append(properties, "VK_QUEUE_VIDEO_DECODE_BIT_KHR")
	}
	if flags&vk.QueueVideoEncodeBit > 0 {
		properties = append(properties, "VK_QUEUE_VIDEO_ENCODE_BIT_KHR")
	}
	if flags&vk.QueueOpticalFlowBitNv > 0 {
		properties = append(properties, "VK_QUEUE_OPTICAL_FLOW_BIT_NV")
	}
	return properties
}
