package main

import "C"
import (
	"GPU_fluid_simulation/model"
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

const MOUSE_SENSITIVITY = 0.5

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
	log.Println("Stating fluid simulation")
	log.Printf("Using GoLang: [%s]", runtime.Version())
}

func onIteration(event sdl.Event, c *Core) {
	switch ev := event.(type) {
	case *sdl.MouseMotionEvent:
		if ev.State == 4 {
			if ev.YRel != 0 {
				yRotAxis := c.cam.LookDir.Cross(c.cam.Up).ScalarMul(-float32(ev.YRel))
				c.cam.Turn(MOUSE_SENSITIVITY, yRotAxis)
			}
			if ev.XRel != 0 {
				xRotAxis := c.cam.Up.ScalarMul(-float32(ev.XRel))
				c.cam.Turn(MOUSE_SENSITIVITY, xRotAxis)
			}
		}
	case *sdl.KeyboardEvent:
		if ev.Type == sdl.KEYUP {
			switch ev.Keysym.Sym {
			case sdl.K_1:
				var newProj int
				if c.cam.ProjectionType == model.CAM_PERSPECTIVE_PROJECTION {
					newProj = model.CAM_ORTHOGRAPHIC_PROJECTION
				} else {
					newProj = model.CAM_PERSPECTIVE_PROJECTION
				}
				log.Printf("Switching projection to -> %d", newProj)
				c.cam.ProjectionType = newProj
			case sdl.K_2:
				if c.cam.LookTarget != nil {
					c.cam.LookTarget = nil
					log.Printf("Free camera resumed at Pos:%v, LookDir:%v", c.cam.Pos, c.cam.LookDir)
				} else {
					c.cam.SetTarget(vm.Vec3{})
					log.Printf("Locked camera to Pos:%v, LookTarget:%v", c.cam.Pos, c.cam.LookTarget)
				}
			case sdl.K_3:
				// Reset camera
				c.cam.Pos = vm.Vec3{Z: -3}
				c.cam.LookDir = vm.Vec3{Z: 1}
				c.cam.LookTarget = nil
				log.Printf("Reset camera to Pos:%v, LookDir:%v", c.cam.Pos, c.cam.LookDir)
			case sdl.K_w:
				c.cam.Move(vm.Vec3{Z: 1})
			case sdl.K_a:
				c.cam.Move(vm.Vec3{X: -1})
			case sdl.K_s:
				c.cam.Move(vm.Vec3{Z: -1})
			case sdl.K_d:
				c.cam.Move(vm.Vec3{X: 1})
			case sdl.K_q:
				c.cam.Turn(10, vm.Vec3{Y: -1})
			case sdl.K_e:
				c.cam.Turn(10, vm.Vec3{Y: 1})
			}
		}
	}
}

func onDraw(elapsed float64, c *Core) {
	m := vm.NewUnitMat(4)

	mod2, err := c.FindInScene("Cube 2")
	mod1, err := c.FindInScene("Cube 1")
	if err != nil {
		log.Println(err)
	} else {
		m, _ = m.Rotate(elapsed*vm.ToRad(45), vm.Vec3{X: 1, Y: 1})
		mod1.Mesh.ModelMat = m
		m, _ = m.Rotate(elapsed*vm.ToRad(20), vm.Vec3{X: 0.5, Y: 1})
		mod2.Mesh.ModelMat = m
	}
}

func main() {
	myModel := model.NewCubeModel("Cube 1")
	myModel2 := model.NewCubeModel("Cube 2")

	core := NewRenderCore()
	core.DefaultCam()
	core.Initialize()
	core.AddToScene(myModel)
	core.AddToScene(myModel2)
	core.loop(
		onIteration,
		onDraw,
	)
	core.ClearScene()
	core.destroy()
}
