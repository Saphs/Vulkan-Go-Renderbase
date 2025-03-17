package renderer

import (
	"errors"
	vk "github.com/goki/vulkan"
	"log"
)

type QueueFamilyIndices struct {
	graphicsFamily *uint32
	presentFamily  *uint32
}

func findQueueFamilies(pd vk.PhysicalDevice, surf vk.Surface) (*QueueFamilyIndices, error) {
	indices := &QueueFamilyIndices{
		graphicsFamily: nil,
		presentFamily:  nil,
	}
	qFamilies := readQueueFamilies(pd)
	//log.Printf("Queue families:\n%s", tableStringQueueFamilyProps(qFamilies))

	// Find first family supporting VK_QUEUE_GRAPHICS_BIT
	for i := range qFamilies {
		if indices.graphicsFamily == nil && isBitSet(qFamilies[i], vk.QueueGraphicsBit) {
			indices.graphicsFamily = new(uint32)
			*indices.graphicsFamily = uint32(i)
		}
		if indices.presentFamily == nil {
			var presentSupport vk.Bool32
			vk.GetPhysicalDeviceSurfaceSupport(pd, uint32(i), surf, &presentSupport)
			if presentSupport > 0 {
				indices.presentFamily = new(uint32)
				*indices.presentFamily = uint32(i)
			}
		}
		if indices.graphicsFamily != nil && indices.presentFamily != nil {
			break
		}
	}
	if indices.graphicsFamily == nil {
		return nil, errors.New("unable to find graphics capable queue family")
	}
	if indices.presentFamily == nil {
		return nil, errors.New("unable to find present capable queue family for given surface")
	}
	return indices, nil
}

func isBitSet(qFamily vk.QueueFamilyProperties, bit vk.QueueFlagBits) bool {
	return vk.QueueFlagBits(qFamily.QueueFlags)&bit > 0
}

func (q *QueueFamilyIndices) isAllQueuesFound() bool {
	return q.graphicsFamily != nil && q.presentFamily != nil
}

func (q *QueueFamilyIndices) toQueueCreateInfos() []vk.DeviceQueueCreateInfo {
	var uniqIndices []uint32
	if q.graphicsFamily == nil {
		log.Panicf("Failed to access graphics capable queue family index")
	}
	if !inList(*q.graphicsFamily, uniqIndices) {
		uniqIndices = append(uniqIndices, *q.graphicsFamily)
	}
	if q.presentFamily == nil {
		log.Panicf("Failed to access present capable queue family index")
	}
	if !inList(*q.presentFamily, uniqIndices) {
		uniqIndices = append(uniqIndices, *q.presentFamily)
	}
	infos := make([]vk.DeviceQueueCreateInfo, len(uniqIndices))
	for i := range uniqIndices {
		infos[i] = vk.DeviceQueueCreateInfo{
			SType:            vk.StructureTypeDeviceQueueCreateInfo,
			PNext:            nil,
			Flags:            0,
			QueueFamilyIndex: uniqIndices[i],
			QueueCount:       1,
			PQueuePriorities: []float32{1.0},
		}
	}
	return infos
}

func inList(e uint32, l []uint32) bool {
	for i := range l {
		if l[i] == e {
			return true
		}
	}
	return false
}
