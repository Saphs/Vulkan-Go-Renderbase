package main

import "C"
import (
	"GPU_fluid_simulation/model"
	"github.com/veandco/go-sdl2/sdl"
	vm "local/vector_math"
	"log"
	"os"
	"runtime"
	"time"
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

const MOV_UNITS_PER_SEC = 5
const MOUSE_SENSITIVITY = 0.5

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
	log.Println("Stating fluid simulation")
	log.Printf("Using GoLang: [%s]", runtime.Version())
}

var dtDraw = time.Now()
var currentlyPressed []sdl.Keycode

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
			removePressedKey(ev.Keysym.Sym)
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
			}
		}
		if ev.Type == sdl.KEYDOWN {
			addPressedKey(ev.Keysym.Sym)
		}
	}
}

func onDraw(elapsed time.Duration, c *Core) {
	drawLast := dtDraw
	dtDraw = time.Now()
	delta := dtDraw.Sub(drawLast)

	m := vm.NewUnitMat(4)
	mod2, err := c.FindInScene("Cube 2")
	mod1, err := c.FindInScene("Cube 1")
	if err != nil {
		log.Println(err)
	} else {
		m, _ = m.Rotate(elapsed.Seconds()*vm.ToRad(45), vm.Vec3{X: 1, Y: 1})
		mod1.Mesh.ModelMat = m
		m, _ = m.Rotate(elapsed.Seconds()*vm.ToRad(20), vm.Vec3{X: 0.5, Y: 1})
		mod2.Mesh.ModelMat = m
	}

	// Interactions with the world that should not happen each event, but each frame
	// ToDo: Introduce third function hook
	// 	-> Non-render relevant things that happen each frame, e.g.: Scene interactions like moving camera
	for _, key := range currentlyPressed {
		switch key {
		case sdl.K_w:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.cam.Move(c.cam.LookDir.ScalarMul(movScale))
		case sdl.K_s:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.cam.Move(c.cam.LookDir.ScalarMul(-movScale))
		case sdl.K_d:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.cam.Move(c.cam.LookDir.Cross(c.cam.Up).ScalarMul(movScale))
		case sdl.K_a:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.cam.Move(c.cam.LookDir.Cross(c.cam.Up).ScalarMul(-movScale))
		case sdl.K_SPACE:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.cam.Move(c.cam.Up.ScalarMul(movScale))
		case sdl.K_LSHIFT:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.cam.Move(c.cam.Up.ScalarMul(-movScale))
		}

	}
}

func addPressedKey(key sdl.Keycode) {
	inList := false
	for i := range currentlyPressed {
		if currentlyPressed[i] == key {
			inList = true
			break
		}
	}
	if !inList {
		currentlyPressed = append(currentlyPressed, key)
	}
}

func removePressedKey(key sdl.Keycode) {
	for i := range currentlyPressed {
		if currentlyPressed[i] == key {
			currentlyPressed[i] = currentlyPressed[len(currentlyPressed)-1]
			currentlyPressed = currentlyPressed[:len(currentlyPressed)-1]
			return
		}
	}
}

func main() {
	myModel := model.NewCubeModel("Cube 1")
	myModel2 := model.NewCubeModel("Cube 2")

	core := NewRenderCore()
	core.DefaultCam()
	core.AddToScene(myModel)
	core.AddToScene(myModel2)
	core.loop(
		onIteration,
		onDraw,
	)
	core.ClearScene()
	core.destroy()
}
