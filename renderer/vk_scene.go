package renderer

import (
	"GPU_fluid_simulation/model"
	"fmt"
	vk "github.com/goki/vulkan"
	vm "local/vector_math"
)

// These functions are part of the rendering core but are split into their own file for logical separation. Their
// focus is scene handling. Adding removing and adjusting things shown in the 3D world of the renderer. In the future
// this could be moved into its own class representing a proper scene tree and its corresponding functionality.

func (c *Core) DefaultCam() {
	cam := model.NewCamera(45, 0.1, 100)
	cam.ProjectionType = model.CAM_PERSPECTIVE_PROJECTION
	cam.Move(vm.Vec3{X: 0, Z: -2})
	c.Cam = cam
}

func (c *Core) FindInScene(name string) (*model.Model, error) {
	for i, v := range c.models {
		if v.Name == name {
			return c.models[i], nil
		}
	}
	return nil, fmt.Errorf("model '%s' not found", name)
}

func (c *Core) AddToScene(model *model.Model) {
	// Careful, we set references for device memory on an object outside the Core.
	// If the object is dereferenced we will not be able to recover this memory
	model.VertexBuffer, model.VertexBufferMem = c.allocateVBuffer(model)
	model.IndexBuffer, model.IndexBufferMem = c.allocateIdxBuffer(model)
	c.models = append(c.models, model)
}

func (c *Core) ClearScene() {
	for _, m := range c.models {
		c.RemoveFromScene(m)
	}
}

// RemoveFromScene drops the reference to a model found in the scene.
// Comparison is done naively by name until more sophisticated methods are required.
func (c *Core) RemoveFromScene(model *model.Model) {
	for i, v := range c.models {
		if v.Name == model.Name {
			vk.DeviceWaitIdle(c.device.D)
			c.DestroyModelBuffers(model)
			c.models = append(c.models[:i], c.models[i+1:]...)
		}
	}
}

func (c *Core) DestroyModelBuffers(model *model.Model) {
	vk.DestroyBuffer(c.device.D, model.VertexBuffer, nil)
	vk.FreeMemory(c.device.D, model.VertexBufferMem, nil)
	vk.DestroyBuffer(c.device.D, model.IndexBuffer, nil)
	vk.FreeMemory(c.device.D, model.IndexBufferMem, nil)
}
