package main

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
const MAX_FRAMES_IN_FLIGHT = 2

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
		if ev.Keysym.Sym == sdl.K_1 {
			log.Printf("Updating vertices")
			c.vertices[0].Pos.X = 1.0
			log.Printf("Now -> %v", c.vertices)
		}
	}
}

func main() {

	// Expected size in memory -> 64 Byte with 4 Bytes of padding as we have 8 Byte words on a 64Bit machine
	v := []vm.Vertex{ // 20 * 3 = 60 Byte
		{ // 8 + 12 = 20 Byte
			Pos:   vm.Vec2{X: -0.5, Y: -0.5}, // 8 Byte (float32 * 2, no padding)
			Color: vm.Vec3{X: 1, Y: 1, Z: 1}, // 12 Byte (float32 * 3, no padding)
		},
		{
			Pos:   vm.Vec2{X: 0.5, Y: -0.5},
			Color: vm.Vec3{X: 0, Y: 1, Z: 0},
		},
		{
			Pos:   vm.Vec2{X: 0.5, Y: 0.5},
			Color: vm.Vec3{X: 0, Y: 0, Z: 1},
		},
		{
			Pos:   vm.Vec2{X: -0.5, Y: 0.5},
			Color: vm.Vec3{X: 0, Y: 0, Z: 1},
		},
	}

	id := []uint32{
		0, 1, 2, 2, 3, 0,
	}

	core := NewRenderCore(v, id)
	core.loop(onIteration)
	core.destroy()
}
