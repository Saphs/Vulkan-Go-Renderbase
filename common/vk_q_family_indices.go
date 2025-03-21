package common

import (
	"errors"
	vk "github.com/goki/vulkan"
	"log"
)

type QueueFamilyIndices struct {
	GraphicsFamily *uint32
	PresentFamily  *uint32
}

func findQueueFamilies(pd vk.PhysicalDevice, surf vk.Surface) (*QueueFamilyIndices, error) {
	indices := &QueueFamilyIndices{
		GraphicsFamily: nil,
		PresentFamily:  nil,
	}
	qFamilies := ReadQueueFamilies(pd)
	//log.Printf("Queue families:\n%s", tableStringQueueFamilyProps(qFamilies))

	// Find first family supporting VK_QUEUE_GRAPHICS_BIT
	for i := range qFamilies {
		if indices.GraphicsFamily == nil && isBitSet(qFamilies[i], vk.QueueGraphicsBit) {
			indices.GraphicsFamily = new(uint32)
			*indices.GraphicsFamily = uint32(i)
		}
		if indices.PresentFamily == nil {
			var presentSupport vk.Bool32
			vk.GetPhysicalDeviceSurfaceSupport(pd, uint32(i), surf, &presentSupport)
			if presentSupport > 0 {
				indices.PresentFamily = new(uint32)
				*indices.PresentFamily = uint32(i)
			}
		}
		if indices.GraphicsFamily != nil && indices.PresentFamily != nil {
			break
		}
	}
	if indices.GraphicsFamily == nil {
		return nil, errors.New("unable to find graphics capable queue family")
	}
	if indices.PresentFamily == nil {
		return nil, errors.New("unable to find present capable queue family for given surface")
	}
	return indices, nil
}

func isBitSet(qFamily vk.QueueFamilyProperties, bit vk.QueueFlagBits) bool {
	return vk.QueueFlagBits(qFamily.QueueFlags)&bit > 0
}

func (q *QueueFamilyIndices) isAllQueuesFound() bool {
	return q.GraphicsFamily != nil && q.PresentFamily != nil
}

func (q *QueueFamilyIndices) toQueueCreateInfos() []vk.DeviceQueueCreateInfo {
	var uniqIndices []uint32
	if q.GraphicsFamily == nil {
		log.Panicf("Failed to access graphics capable queue family index")
	}
	if !inList(*q.GraphicsFamily, uniqIndices) {
		uniqIndices = append(uniqIndices, *q.GraphicsFamily)
	}
	if q.PresentFamily == nil {
		log.Panicf("Failed to access present capable queue family index")
	}
	if !inList(*q.PresentFamily, uniqIndices) {
		uniqIndices = append(uniqIndices, *q.PresentFamily)
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
