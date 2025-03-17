package main

import "C"
import (
	"GPU_fluid_simulation/model"
	"GPU_fluid_simulation/renderer"
	"github.com/veandco/go-sdl2/sdl"
	vm "local/vector_math"
	"log"
	"os"
	"runtime"
	"time"
)

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

func onIteration(event sdl.Event, c *renderer.Core) {
	switch ev := event.(type) {
	case *sdl.MouseMotionEvent:
		if ev.State == 4 {
			if ev.YRel != 0 {
				yRotAxis := c.Cam.LookDir.Cross(c.Cam.Up).ScalarMul(-float32(ev.YRel))
				c.Cam.Turn(MOUSE_SENSITIVITY, yRotAxis)
			}
			if ev.XRel != 0 {
				xRotAxis := c.Cam.Up.ScalarMul(-float32(ev.XRel))
				c.Cam.Turn(MOUSE_SENSITIVITY, xRotAxis)
			}
		}
	case *sdl.KeyboardEvent:
		if ev.Type == sdl.KEYUP {
			removePressedKey(ev.Keysym.Sym)
			switch ev.Keysym.Sym {
			case sdl.K_1:
				var newProj int
				if c.Cam.ProjectionType == model.CAM_PERSPECTIVE_PROJECTION {
					newProj = model.CAM_ORTHOGRAPHIC_PROJECTION
				} else {
					newProj = model.CAM_PERSPECTIVE_PROJECTION
				}
				log.Printf("Switching projection to -> %d", newProj)
				c.Cam.ProjectionType = newProj
			case sdl.K_2:
				if c.Cam.LookTarget != nil {
					c.Cam.LookTarget = nil
					log.Printf("Free camera resumed at Pos:%v, LookDir:%v", c.Cam.Pos, c.Cam.LookDir)
				} else {
					c.Cam.SetTarget(vm.Vec3{})
					log.Printf("Locked camera to Pos:%v, LookTarget:%v", c.Cam.Pos, c.Cam.LookTarget)
				}
			case sdl.K_3:
				// Reset camera
				c.Cam.Pos = vm.Vec3{Z: -3}
				c.Cam.LookDir = vm.Vec3{Z: 1}
				c.Cam.LookTarget = nil
				log.Printf("Reset camera to Pos:%v, LookDir:%v", c.Cam.Pos, c.Cam.LookDir)
			}
		}
		if ev.Type == sdl.KEYDOWN {
			addPressedKey(ev.Keysym.Sym)
		}
	}
}

func onDraw(elapsed time.Duration, c *renderer.Core) {
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
			c.Cam.Move(c.Cam.LookDir.ScalarMul(movScale))
		case sdl.K_s:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.Cam.Move(c.Cam.LookDir.ScalarMul(-movScale))
		case sdl.K_d:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.Cam.Move(c.Cam.LookDir.Cross(c.Cam.Up).ScalarMul(movScale))
		case sdl.K_a:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.Cam.Move(c.Cam.LookDir.Cross(c.Cam.Up).ScalarMul(-movScale))
		case sdl.K_SPACE:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.Cam.Move(c.Cam.Up.ScalarMul(movScale))
		case sdl.K_LSHIFT:
			movScale := float32(delta.Seconds()) * MOV_UNITS_PER_SEC
			c.Cam.Move(c.Cam.Up.ScalarMul(-movScale))
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

	core := renderer.NewRenderCore()
	core.DefaultCam()
	core.AddToScene(myModel)
	core.AddToScene(myModel2)
	core.Loop(
		onIteration,
		onDraw,
	)
	core.ClearScene()
	core.Destroy()
}
