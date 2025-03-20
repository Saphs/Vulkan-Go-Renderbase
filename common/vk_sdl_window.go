package common

import (
	"fmt"
	vk "github.com/goki/vulkan"
	"github.com/veandco/go-sdl2/sdl"
	"log"
)

type Window struct {
	sdlVersion string
	vkVersion  string

	win       *sdl.Window
	Resized   bool
	Minimized bool
	Close     bool

	inst *vk.Instance
	surf *vk.Surface
}

func NewWindow(title string, w int32, h int32, validationLayers []string) *Window {
	window := &Window{
		sdlVersion: fmt.Sprintf("v%d.%d.%d", sdl.MAJOR_VERSION, sdl.MINOR_VERSION, sdl.PATCHLEVEL),
		// go bindings v1.0.7 -> Vulkan spec, as per: https://github.com/goki/vulkan = 1.3.239
		vkVersion: "v1.3.239",

		Resized:   false,
		Minimized: false,
		Close:     false,
	}
	window.initSDLWindow(title, w, h)
	window.initVulkan()
	window.createVulkanInstance(len(validationLayers) > 0, validationLayers)
	log.Printf("Generated SDL/Vulkan window - SDL: %s Vulkan Spec: %s", window.sdlVersion, window.vkVersion)
	return window
}

func (w *Window) Destroy() {
	vk.DestroyInstance(*w.inst, nil)
	err := w.win.Destroy()
	if err != nil {
		log.Fatal(err)
	}
}

func (w *Window) initSDLWindow(title string, width int32, height int32) *sdl.Window {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic("Failed to initialize a SDL !")
	}
	log.Println("Creating SDL window")
	win, err := sdl.CreateWindow(
		title,
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		width,
		height,
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE|sdl.WINDOW_VULKAN,
	)
	if err != nil {
		panic(err)
	}
	return win
}

func (w *Window) initVulkan() {
	vk.SetGetInstanceProcAddr(sdl.VulkanGetVkGetInstanceProcAddr())
	err := vk.Init()
	if err != nil {
		panic(err)
	}
}

func (w *Window) createVulkanInstance(enableValidation bool, validationLayers []string) {
	requiredExtensions := w.win.VulkanGetInstanceExtensions()
	checkInstanceExtensionSupport(requiredExtensions)

	if enableValidation {
		log.Printf("Validation enabled, checking layer support")
		checkValidationLayerSupport(validationLayers)
	}
	applicationInfo := &vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		PNext:              nil,
		PApplicationName:   "GPU fluid simulation",
		ApplicationVersion: vk.MakeVersion(1, 0, 0),
		PEngineName:        "No Engine",
		EngineVersion:      vk.MakeVersion(1, 0, 0),
		ApiVersion:         vk.MakeVersion(1, 0, 0),
	}
	createInfo := &vk.InstanceCreateInfo{
		SType:                   vk.StructureTypeInstanceCreateInfo,
		PNext:                   nil,
		Flags:                   0,
		PApplicationInfo:        applicationInfo,
		EnabledLayerCount:       0,
		PpEnabledLayerNames:     nil,
		EnabledExtensionCount:   uint32(len(requiredExtensions)),
		PpEnabledExtensionNames: TerminatedStrs(requiredExtensions),
	}
	if enableValidation {
		createInfo.EnabledLayerCount = uint32(len(validationLayers))
		createInfo.PpEnabledLayerNames = TerminatedStrs(validationLayers)
	}
	ins, err := VkCreateInstance(createInfo, nil)
	if err != nil {
		log.Panicf("Failed to create vk instance, due to: %v", err)
	}
	w.inst = &ins
}

func checkInstanceExtensionSupport(requiredInstanceExt []string) {
	supportedExtNames := ReadInstanceExtensionPropertyNames()
	log.Printf("Required instance extensions: %v", requiredInstanceExt)
	log.Printf("Available extensions (%d): %v", len(supportedExtNames), supportedExtNames)

	if !AllOfAinB(requiredInstanceExt, supportedExtNames) {
		log.Panicf("At least one required instance extension is not supported")
	} else {
		log.Println("Success - All required instance extensions are supported")
	}
}

func checkValidationLayerSupport(requiredLayers []string) {
	supportedLayerNames := ReadInstanceLayerPropertyNames()
	log.Printf("Desired validation layers: %v", requiredLayers)
	log.Printf("Supported layers (%d): %v", len(supportedLayerNames), supportedLayerNames)

	if !AllOfAinB(requiredLayers, supportedLayerNames) {
		log.Panicf("At least one desired layers are supported")
	} else {
		log.Println("Success - All desired validation layers are supported")
	}
}
