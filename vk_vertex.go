package main

import (
	vk "github.com/goki/vulkan"
	"local/vector_math"
	"unsafe"
)

type Vertex struct {
	Pos   vector_math.Vec3
	Color vector_math.Vec3
}

func GetVertexBindingDescription() vk.VertexInputBindingDescription {
	return vk.VertexInputBindingDescription{
		Binding:   0,
		Stride:    uint32(unsafe.Sizeof(Vertex{})),
		InputRate: vk.VertexInputRateVertex,
	}
}

func GetVertexAttributeDescriptions() []vk.VertexInputAttributeDescription {
	return []vk.VertexInputAttributeDescription{
		{
			Location: 0,
			Binding:  0,
			Format:   vk.FormatR32g32b32Sfloat,
			Offset:   uint32(unsafe.Offsetof(Vertex{}.Pos)),
		},
		{
			Location: 1,
			Binding:  0,
			Format:   vk.FormatR32g32b32Sfloat,
			Offset:   uint32(unsafe.Offsetof(Vertex{}.Color)),
		},
	}
}
