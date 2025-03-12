package model

import (
	"GPU_fluid_simulation/tooling"
	vk "github.com/goki/vulkan"
	vm "local/vector_math"
	"unsafe"
)

type Model struct {
	Mesh            *Mesh
	Name            string
	VertexBuffer    vk.Buffer
	VertexBufferMem vk.DeviceMemory
	IndexBuffer     vk.Buffer
	IndexBufferMem  vk.DeviceMemory
}

func NewModel(m *Mesh, n string) *Model {
	return &Model{
		Name: n,
		Mesh: m,
	}
}

// ModelPushConstantsSize reports the memory size required for all push constants that the Model expects to
// get bound. The actual layout for the constants in memory is decided by the render pipeline. For now only
// the Mesh.ModelMat (4x4) needs to be provided.
func ModelPushConstantsSize() uint32 {
	mat := vm.NewUnitMat(4)
	return uint32(mat.ByteSize())
}

// GetVBufferSize returns the size required for keeping this model in device memory.
// Mainly used to determine the buffer size when calling in Code.createBuffer(size vk.DeviceSize, ...)
func (m *Model) GetVBufferSize() int {
	// ToDo: Fix non-performant workaround, the calculation of the size should be fast and simple but the old on didnt work
	// old one -> return int(unsafe.Sizeof(m.Mesh.Vertices)) * len(m.Mesh.Vertices)
	return len(tooling.RawBytes(m.Mesh.Vertices))
}

// GetVBufferBytes returns the raw bytes representing all vertices for this model.
// Mainly used to execute vk.Memcopy(..., src []byte) to move memory from CPU to GPU
func (m *Model) GetVBufferBytes() []byte {
	return tooling.RawBytes(m.Mesh.Vertices)
}

// GetIdxBufferSize returns the size required for keeping this model in device memory.
// Mainly used to determine the buffer size when calling in Code.createBuffer(size vk.DeviceSize, ...)
func (m *Model) GetIdxBufferSize() int {
	return int(unsafe.Sizeof(m.Mesh.VIndices[0])) * len(m.Mesh.VIndices)
}

// GetIdxBufferBytes returns the raw bytes representing the indices used to address vertex data for this model.
// Mainly used to execute vk.Memcopy(..., src []byte) to move memory from CPU to GPU
func (m *Model) GetIdxBufferBytes() []byte {
	return tooling.RawBytes(m.Mesh.VIndices)
}
