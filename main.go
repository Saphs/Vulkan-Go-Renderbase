package main

import (
	"github.com/veandco/go-sdl2/sdl"
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
	}
}

func main() {
	core := NewRenderCore()
	core.loop(onIteration)
	core.destroy()
}
