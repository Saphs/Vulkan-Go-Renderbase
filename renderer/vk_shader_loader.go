package renderer

import (
	"GPU_fluid_simulation/common"
	vk "github.com/goki/vulkan"
	"log"
	"os"
)

// LoadVert reads a '.spv' file with the expectation of it containing a vertex shader for later use in a
// render pipeline. For this, a shader module (containing the shader code) and its vk.PipelineShaderStageCreateInfo
// is returned. Which is required to bind the shader to the pipeline.
func LoadVert(d vk.Device, path string) (vk.ShaderModule, vk.PipelineShaderStageCreateInfo) {
	vertMod := readShaderCode(d, path)
	log.Printf("Created vertex shader module: %v", vertMod)

	vertexShaderStageInfo := vk.PipelineShaderStageCreateInfo{
		SType:               vk.StructureTypePipelineShaderStageCreateInfo,
		PNext:               nil,
		Flags:               0,
		Stage:               vk.ShaderStageVertexBit,
		Module:              vertMod,
		PName:               "main\x00", // entrypoint -> function name in the shader
		PSpecializationInfo: nil,
	}
	return vertMod, vertexShaderStageInfo
}

// LoadFrag reads a '.spv' file with the expectation of it containing a fragment shader for later use in a
// render pipeline. For this, a shader module (containing the shader code) and its vk.PipelineShaderStageCreateInfo
// is returned. Which is required to bind the shader to the pipeline.
func LoadFrag(d vk.Device, path string) (vk.ShaderModule, vk.PipelineShaderStageCreateInfo) {
	fragMod := readShaderCode(d, path)
	log.Printf("Created fragment shader module: %v", fragMod)

	fragmentShaderStageInfo := vk.PipelineShaderStageCreateInfo{
		SType:               vk.StructureTypePipelineShaderStageCreateInfo,
		PNext:               nil,
		Flags:               0,
		Stage:               vk.ShaderStageFragmentBit,
		Module:              fragMod,
		PName:               "main\x00", // entrypoint -> function name in the shader
		PSpecializationInfo: nil,
	}
	return fragMod, fragmentShaderStageInfo
}

// DeleteShaderMod discards a shader module. As vk.ShaderModule is only meant as a container to move the shader code
// onto device memory, it can be destroyed right after creating a shader stage when binding to a rendering pipeline.
func DeleteShaderMod(d vk.Device, mod vk.ShaderModule) {
	vk.DestroyShaderModule(d, mod, nil)
}

func readShaderCode(d vk.Device, shaderFile string) vk.ShaderModule {
	shaderCodeB, err := os.ReadFile(shaderFile)
	shaderCodeLen := uint64(len(shaderCodeB))
	if err != nil {
		log.Panicf("Failed to read shader file: '%s' due to: %v", shaderFile, err)
	}
	log.Printf("Read shader file (%s) of size: %dByte", shaderFile, shaderCodeLen)

	createInfo := &vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		PNext:    nil,
		Flags:    0,
		CodeSize: shaderCodeLen,
		PCode:    common.AsUint32Arr(shaderCodeB),
	}
	module, err := common.VKCreateShaderModule(d, createInfo, nil)
	if err != nil {
		log.Panicf("Failed to create shader module: '%s'", shaderFile)
	}
	return module
}
