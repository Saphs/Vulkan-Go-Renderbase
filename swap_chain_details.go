package main

import (
	vk "github.com/goki/vulkan"
	"log"
)

type SwapChainDetails struct {
	capabilities vk.SurfaceCapabilities
	formats      []vk.SurfaceFormat
	presentModes []vk.PresentMode
}

func (s *SwapChainDetails) selectSwapSurfaceFormat() vk.SurfaceFormat {
	for _, af := range s.formats {
		if af.Format == vk.FormatB8g8r8a8Srgb && af.ColorSpace == vk.ColorSpaceSrgbNonlinear {
			return af
		}
	}
	fallbackFormat := s.formats[0]
	log.Printf("Did not find prefered SurfaceFormat, selecting first one available. (%v)", fallbackFormat)
	return fallbackFormat
}

func (s *SwapChainDetails) selectSwapPresentMode() vk.PresentMode {
	for _, pm := range s.presentModes {
		if pm == vk.PresentModeMailbox {
			return pm
		}
	}
	fallbackMode := vk.PresentModeFifo
	log.Printf("Did not find prefered PresentMode, selecting FIFO. (%v)", fallbackMode)
	return fallbackMode
}

func (s *SwapChainDetails) chooseSwapExtent() vk.Extent2D {
	// Returning the current extend directly as I dont want to do anything crazy and
	// https://github.com/vulkan-go/demos/blob/master/vulkandraw/vulkandraw.go does the same
	// I can return to this later: https://vulkan-tutorial.com/Drawing_a_triangle/Presentation/Swap_chain
	s.capabilities.CurrentExtent.Deref()
	return s.capabilities.CurrentExtent
}
