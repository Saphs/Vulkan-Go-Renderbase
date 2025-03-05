package main

import (
	"errors"
	vk "github.com/goki/vulkan"
	"github.com/veandco/go-sdl2/sdl"
	"log"
)

// Utility functions wrapping the raw go bindings to provide a more go-lang style interface. This should not
// hide or alter behavior and only allow for more tidy core code by tweaking signatures.

func VkCreateInstance(pCreateInfo *vk.InstanceCreateInfo, pAllocator *vk.AllocationCallbacks) (vk.Instance, error) {
	var in vk.Instance
	err := vk.Error(vk.CreateInstance(pCreateInfo, pAllocator, &in))
	if err != nil {
		return nil, err
	}
	err = vk.InitInstance(in)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func sdlCreateVkSurface(win *sdl.Window, instance vk.Instance) (vk.Surface, error) {
	surfPtr, err := win.VulkanCreateSurface(instance)
	if err != nil {
		return nil, err
	}
	return vk.SurfaceFromPointer(uintptr(surfPtr)), nil
}

func VkCreateDevice(physicalDevice vk.PhysicalDevice, pCreateInfo *vk.DeviceCreateInfo, pAllocator *vk.AllocationCallbacks) (vk.Device, error) {
	var d vk.Device
	err := vk.Error(vk.CreateDevice(physicalDevice, pCreateInfo, pAllocator, &d))
	if err != nil {
		return nil, err
	}
	return d, nil
}

func VkGetDeviceQueue(device vk.Device, queueFamilyIndex *uint32, queueIndex uint32) (vk.Queue, error) {
	var q vk.Queue
	if queueFamilyIndex == nil {
		return nil, errors.New("QueueFamily index was nil")
	}
	vk.GetDeviceQueue(device, *queueFamilyIndex, queueIndex, &q)
	return q, nil
}

func VkCreateSwapChain(device vk.Device, pCreateInfo *vk.SwapchainCreateInfo, pAllocator *vk.AllocationCallbacks) (vk.Swapchain, error) {
	var sc vk.Swapchain
	err := vk.Error(vk.CreateSwapchain(device, pCreateInfo, pAllocator, &sc))
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func VkCreateImageView(device vk.Device, pCreateInfo *vk.ImageViewCreateInfo, pAllocator *vk.AllocationCallbacks) (vk.ImageView, error) {
	var iv vk.ImageView
	err := vk.Error(vk.CreateImageView(device, pCreateInfo, pAllocator, &iv))
	if err != nil {
		return nil, err
	}
	return iv, nil
}

func VkCreateRenderPass(device vk.Device, pCreateInfo *vk.RenderPassCreateInfo, pAllocator *vk.AllocationCallbacks) (vk.RenderPass, error) {
	var pr vk.RenderPass
	err := vk.Error(vk.CreateRenderPass(device, pCreateInfo, pAllocator, &pr))
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func VkCreateFrameBuffer(device vk.Device, pCreateInfo *vk.FramebufferCreateInfo, pAllocator *vk.AllocationCallbacks) (vk.Framebuffer, error) {
	var fb vk.Framebuffer
	err := vk.Error(vk.CreateFramebuffer(device, pCreateInfo, pAllocator, &fb))
	if err != nil {
		return nil, err
	}
	return fb, nil
}

func VkCreatePipelineLayout(device vk.Device, pCreateInfo *vk.PipelineLayoutCreateInfo, pAllocator *vk.AllocationCallbacks) (vk.PipelineLayout, error) {
	var pl vk.PipelineLayout
	err := vk.Error(vk.CreatePipelineLayout(device, pCreateInfo, pAllocator, &pl))
	if err != nil {
		return nil, err
	}
	return pl, nil
}

func VkCreateGraphicsPipelines(device vk.Device, pipelineCache vk.PipelineCache, createInfoCount uint32, pCreateInfos []vk.GraphicsPipelineCreateInfo, pAllocator *vk.AllocationCallbacks) ([]vk.Pipeline, error) {
	var gp = make([]vk.Pipeline, createInfoCount)
	err := vk.Error(vk.CreateGraphicsPipelines(device, pipelineCache, createInfoCount, pCreateInfos, pAllocator, gp))
	if err != nil {
		return nil, err
	}
	return gp, nil
}

func VkCreateCommandPool(device vk.Device, pCreateInfo *vk.CommandPoolCreateInfo, pAllocator *vk.AllocationCallbacks) (vk.CommandPool, error) {
	var cp vk.CommandPool
	err := vk.Error(vk.CreateCommandPool(device, pCreateInfo, pAllocator, &cp))
	if err != nil {
		return nil, err
	}
	log.Printf("Successfully created command pool")
	return cp, nil
}
