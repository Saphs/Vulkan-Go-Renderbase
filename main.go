package main

import "C"
import (
	"github.com/veandco/go-sdl2/sdl"
	vm "local/vector_math"
	"log"
	"os"
	"runtime"
)

const ENABLE_VALIDATION = true

var VALIDATION_LAYERS = []string{
	"VK_LAYER_KHRONOS_validation",
}

var DEVICE_EXTENSIONS = []string{
	"VK_KHR_swapchain",
}

const PROGRAM_NAME = "GPU fluid simulation"
const WINDOW_WIDTH, WINDOW_HEIGHT int32 = 1280, 720
const MAX_FRAMES_IN_FLIGHT = 3

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
	log.Println("Stating fluid simulation")
	log.Printf("Using GoLang: [%s]", runtime.Version())
}

func onIteration(event sdl.Event, c *Core) {
	switch ev := event.(type) {
	case *sdl.MouseMotionEvent:
		log.Printf(
			"[%d ms] MouseMotion\tid:%d\tx:%d\ty:%d\txrel:%d\tyrel:%d\n",
			ev.Timestamp,
			ev.Which,
			ev.X,
			ev.Y,
			ev.XRel,
			ev.YRel,
		)
	case *sdl.KeyboardEvent:
		if ev.Type == sdl.KEYUP {
			switch ev.Keysym.Sym {
			case sdl.K_1:
				var newProj int
				if c.cam.ProjectionType == vm.CAM_PERSPECTIVE_PROJECTION {
					newProj = vm.CAM_ORTHOGRAPHIC_PROJECTION
				} else {
					newProj = vm.CAM_PERSPECTIVE_PROJECTION
				}
				log.Printf("Switching projection to -> %d", newProj)
				c.cam.ProjectionType = newProj
			case sdl.K_2:
				if c.cam.LookTarget != nil {
					c.cam.LookTarget = nil
				} else {
					c.cam.SetTarget(vm.Vec3{})
				}
			case sdl.K_3:
				// Reset camera
				c.cam.Pos = vm.Vec3{Z: -3}
				c.cam.LookDir = vm.Vec3{Z: 1}
				c.cam.LookTarget = nil
			case sdl.K_w:
				c.cam.Move(vm.Vec3{Z: 1})
			case sdl.K_a:
				c.cam.Move(vm.Vec3{X: -1})
			case sdl.K_s:
				c.cam.Move(vm.Vec3{Z: -1})
			case sdl.K_d:
				c.cam.Move(vm.Vec3{X: 1})
			}
		}
	}
}

func onDraw(elapsed float64, c *Core) {
	m := vm.NewUnitMat(4)
	m, _ = m.Rotate(elapsed*vm.ToRad(45), vm.Vec3{X: 1, Y: 1})
	c.mesh.ModelMat = m
}

func main() {

	// Expected size in memory -> 64 Byte with 4 Bytes of padding as we have 8 Byte words on a 64Bit machine
	v := []vm.Vertex{ // 24 * 8 = 192 Byte
		{ // 8 + 12 = 24 Byte [0]
			Pos:   vm.Vec3{X: -0.5, Y: -0.5, Z: -0.5}, // 12 Byte (float32 * 3, no padding)
			Color: vm.Vec3{X: 1, Y: 0, Z: 0},          // 12 Byte (float32 * 3, no padding)
		},
		{ // [1]
			Pos:   vm.Vec3{X: 0.5, Y: -0.5, Z: -0.5},
			Color: vm.Vec3{X: 0, Y: 1, Z: 0},
		},
		{ // [2]
			Pos:   vm.Vec3{X: 0.5, Y: 0.5, Z: -0.5},
			Color: vm.Vec3{X: 0, Y: 0, Z: 1},
		},
		{ // [3]
			Pos:   vm.Vec3{X: -0.5, Y: 0.5, Z: -0.5},
			Color: vm.Vec3{X: 1, Y: 0.5, Z: 1},
		},
		{ // 8 + 12 = 20 Byte [4]
			Pos:   vm.Vec3{X: -0.5, Y: -0.5, Z: 0.5}, // 12 Byte (float32 * 3, no padding)
			Color: vm.Vec3{X: 1, Y: 0.5, Z: 0.5},     // 12 Byte (float32 * 3, no padding)
		},
		{ // [5]
			Pos:   vm.Vec3{X: 0.5, Y: -0.5, Z: 0.5},
			Color: vm.Vec3{X: 0.5, Y: 1, Z: 0.5},
		},
		{ // [6]
			Pos:   vm.Vec3{X: 0.5, Y: 0.5, Z: 0.5},
			Color: vm.Vec3{X: 0.5, Y: 0.5, Z: 1},
		},
		{ // [7]
			Pos:   vm.Vec3{X: -0.5, Y: 0.5, Z: 0.5},
			Color: vm.Vec3{X: 0, Y: 0.5, Z: 0},
		},
	}

	id := []uint32{
		2, 1, 0, 0, 3, 2, // front
		5, 1, 6, 1, 2, 6, // right
		4, 5, 6, 7, 4, 6, // back
		4, 7, 0, 0, 7, 3, // left
		0, 1, 5, 5, 4, 0, // top
		3, 7, 6, 2, 3, 6, // bottom
	}

	cam := vm.NewCamera(45, 0.1, 100)
	cam.ProjectionType = vm.CAM_PERSPECTIVE_PROJECTION
	cam.View = vm.NewDirectionView(
		vm.Vec3{X: 0, Z: -5},
		vm.Vec3{0, 0, 5},
		vm.Vec3{Y: -1},
	)

	mesh := vm.NewMesh(v, id)
	mesh.ModelMat, _ = mesh.ModelMat.Translate(vm.Vec3{
		X: 0,
		Y: 0,
		Z: 5,
	})

	core := NewRenderCore()
	core.SetScene(mesh, cam)
	core.Initialize()
	core.loop(
		onIteration,
		onDraw,
	)
	core.destroy()
}
