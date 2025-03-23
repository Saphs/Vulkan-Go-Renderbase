package common

import (
	"fmt"
	vk "github.com/goki/vulkan"
	"github.com/veandco/go-sdl2/sdl"
	"log"
)

const APPLICATION_NAME = "GPU fluid simulation"
const APP_MAJOR, APP_MINOR, APP_PATCH = 1, 0, 0
const ENGINE_NAME = "No Engine"
const ENGINE_MAJOR, ENGINE_MINOR, ENGINE_PATCH = 1, 0, 0

const SDL_MAJOR, SDL_MINOR, SDL_PATCH = int(sdl.MAJOR_VERSION), int(sdl.MINOR_VERSION), int(sdl.PATCHLEVEL)

// Vulkan spec go bindings = v1.0.7, as per: https://github.com/goki/vulkan = 1.3.239
const VK_SPEC_MAJOR, VK_SPEC_MINOR, VK_SPEC_PATCH int = 1, 3, 239

// Window encapsulates all window handling components and vulkan access objects to talk, to actual draw on screen. It
// uses SDL for window management and user input, for a Vulkan application. Thus simplifying the process of getting a
// vk.surface to draw on and interact with.
type Window struct {
	sdlVersion string
	vkVersion  string

	Win       *sdl.Window
	Resized   bool
	Minimized bool
	Close     bool

	Inst *vk.Instance
	Surf *vk.Surface
}

// NewWindow constructs a new Window struct by default initializing things, stating some meta information and
// calling the corresponding init functions for the SDL window, Vulkan API instance and so on. On tear down,
// we need to destroy the: vk.surface, vk.instance and sdl.window.
func NewWindow(title string, w int32, h int32, validationLayers []string) *Window {
	window := &Window{
		sdlVersion: fmt.Sprintf("v%d.%d.%d", SDL_MAJOR, SDL_MINOR, SDL_PATCH),
		vkVersion:  fmt.Sprintf("v%d.%d.%d", VK_SPEC_MAJOR, VK_SPEC_MINOR, VK_SPEC_PATCH),
		Resized:    false,
		Minimized:  false,
		Close:      false,
	}
	window.initSDLWindow(title, w, h)
	window.initVulkan()
	window.createVulkanInstance(len(validationLayers) > 0, validationLayers)
	window.createSdlVkSurface()
	log.Printf("Generated SDL/Vulkan window - SDL: %s Vulkan Spec: %s", window.sdlVersion, window.vkVersion)
	return window
}

// Destroy is a convenience method to tear down all relevant instances (vk.surface, vk.instance and sdl.window)
// that have been initialized by itself.
func (w *Window) Destroy() {
	vk.DestroySurface(*w.Inst, *w.Surf, nil)
	vk.DestroyInstance(*w.Inst, nil)
	err := w.Win.Destroy()
	if err != nil {
		log.Fatal(err)
	}
}

func (w *Window) initSDLWindow(title string, width int32, height int32) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Panicf("Failed to initialize SDL: %v", err)
	}
	log.Println("Initialized SDL")
	win, err := sdl.CreateWindow(
		title,
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		width,
		height,
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE|sdl.WINDOW_VULKAN,
	)
	if err != nil {
		log.Panicf("Failed to create SDL window for use with Vulkan: %v", err)
	}
	log.Printf("Created SDL window for use with Vulkan. Title: \"%s\", Width: %d, Height: %d", title, width, height)
	w.Win = win
}

func (w *Window) initVulkan() {
	// Find and load Vulkan addresses to be able to call driver level functions via provided mechanism
	vk.SetGetInstanceProcAddr(sdl.VulkanGetVkGetInstanceProcAddr())
	err := vk.Init()
	if err != nil {
		log.Panicf("Failed to initialize Vulkan API: %v", err)
	}
}

func (w *Window) createVulkanInstance(enableValidation bool, validationLayers []string) {
	requiredExtensions := w.Win.VulkanGetInstanceExtensions()
	checkInstanceExtensionSupport(requiredExtensions)

	if enableValidation {
		log.Printf("Validation enabled, checking layer support")
		checkValidationLayerSupport(validationLayers)
	}
	applicationInfo := &vk.ApplicationInfo{
		SType:              vk.StructureTypeApplicationInfo,
		PNext:              nil,
		PApplicationName:   APPLICATION_NAME,
		ApplicationVersion: vk.MakeVersion(APP_MAJOR, APP_MINOR, APP_PATCH),
		PEngineName:        ENGINE_NAME,
		EngineVersion:      vk.MakeVersion(ENGINE_MAJOR, ENGINE_MINOR, ENGINE_PATCH),
		ApiVersion:         vk.MakeVersion(VK_SPEC_MAJOR, VK_SPEC_MINOR, VK_SPEC_PATCH),
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
	w.Inst = &ins
}

func checkInstanceExtensionSupport(requiredInstanceExt []string) {
	supportedExtNames := ReadInstanceExtensionPropertyNames()
	log.Printf("Required instance extensions: %v", requiredInstanceExt)
	log.Printf("Available extensions (%d): %v", len(supportedExtNames), supportedExtNames)

	if !IsSubset(requiredInstanceExt, supportedExtNames) {
		log.Panicf("At least one required instance extension is not supported")
	} else {
		log.Println("Success - All required instance extensions are supported")
	}
}

func checkValidationLayerSupport(requiredLayers []string) {
	supportedLayerNames := ReadInstanceLayerPropertyNames()
	log.Printf("Desired validation layers: %v", requiredLayers)
	log.Printf("Supported layers (%d): %v", len(supportedLayerNames), supportedLayerNames)

	if !IsSubset(requiredLayers, supportedLayerNames) {
		log.Panicf("At least one desired layers are supported")
	} else {
		log.Println("Success - All desired validation layers are supported")
	}
}

func (w *Window) createSdlVkSurface() {
	surf, err := SdlCreateVkSurface(w.Win, *w.Inst)
	if err != nil {
		log.Panicf("Failed to create SDL window's Vulkan-surface, due to: %v", err)
	}
	w.Surf = &surf
}
