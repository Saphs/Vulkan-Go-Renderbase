package renderer

import (
	com "GPU_fluid_simulation/common"
	"GPU_fluid_simulation/model"
	"fmt"
	vk "github.com/goki/vulkan"
	vm "local/vector_math"
	"log"
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

func (c *Core) AddToScene(m *model.Model) {

	// Careful, we set references for device memory on an object outside the Core.
	// If the object is dereferenced we will not be able to recover this memory
	m.VertexBuffer, m.VertexBufferMem = c.allocateVBuffer(m)
	m.IndexBuffer, m.IndexBufferMem = c.allocateIdxBuffer(m)
	c.models = append(c.models, m)
}

// ClearScene gracefully removes one object at a time expecting the RemoveFromScene function to never fail
func (c *Core) ClearScene() {
	log.Printf("Clear scene")
	for i := len(c.models) - 1; i >= 0; i-- {
		c.RemoveFromScene(c.models[i])
	}
}

// ClearSceneForced clears the scene from any objects still in the model list, freeing everything it can.
// This disregards any expectations on what is removed
func (c *Core) ClearSceneForced() {
	log.Printf("Forcully emptying the scene of %d models", len(c.models))
	for i := len(c.models) - 1; i >= 0; i-- {
		err := com.VKDeviceWaitIdle(c.device.D)
		if err != nil {
			log.Panicf("Failed to wait on device idle to forcefully clear scene: %v", err)
		}
		c.DestroyModelBuffers(c.models[i])
		c.models[i] = nil
	}
	c.models = c.models[:0]
}

// RemoveFromScene drops the reference to a model found in the scene.
// Comparison is done naively by name until more sophisticated methods are required.
func (c *Core) RemoveFromScene(model *model.Model) {
	idx := -1
	for i, v := range c.models {
		if v.Name == model.Name {
			idx = i
			log.Printf("Found model to remove '%s' %v", model.Name, model)
			break
		}
	}
	if idx == -1 {
		log.Printf("Unable to find model to remove '%s'", model.Name)
		return
	}
	err := com.VKDeviceWaitIdle(c.device.D)
	if err != nil {
		log.Panicf("Failed to wait on device idle remove model: %v", err)
	}
	c.DestroyModelBuffers(model)
	// Generic delete from slice: https://go.dev/wiki/SliceTricks
	c.models[idx] = c.models[len(c.models)-1]
	c.models[len(c.models)-1] = nil
	c.models = c.models[:len(c.models)-1]
}

func (c *Core) DestroyModelBuffers(model *model.Model) {
	vk.DestroyBuffer(c.device.D, model.VertexBuffer, nil)
	vk.FreeMemory(c.device.D, model.VertexBufferMem, nil)
	vk.DestroyBuffer(c.device.D, model.IndexBuffer, nil)
	vk.FreeMemory(c.device.D, model.IndexBufferMem, nil)
}
