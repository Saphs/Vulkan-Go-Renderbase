package renderer

import (
	com "GPU_fluid_simulation/common"
	"GPU_fluid_simulation/model"
	"log"

	vk "github.com/goki/vulkan"
)

type DescriptorProvisioner struct {
	device vk.Device

	descriptorSetLayout vk.DescriptorSetLayout
	descriptorPool      vk.DescriptorPool
	descriptorSets      []vk.DescriptorSet

	modelDescriptorSetLayout vk.DescriptorSetLayout
	modelDescriptorPool      vk.DescriptorPool
	modelDescriptorSets      []vk.DescriptorSet
}

func NewDescriptorProvisioner(device vk.Device) *DescriptorProvisioner {
	return &DescriptorProvisioner{
		device: device,
	}
}

// allocDescriptorSets Allocates a list of descriptor sets of given layout from the stated pool
func (dp *DescriptorProvisioner) allocDescriptorSets(pool vk.DescriptorPool, layouts []vk.DescriptorSetLayout) []vk.DescriptorSet {
	cnt := uint32(len(layouts))
	allocInfo := vk.DescriptorSetAllocateInfo{
		SType:              vk.StructureTypeDescriptorSetAllocateInfo,
		PNext:              nil,
		DescriptorPool:     pool,
		DescriptorSetCount: cnt,
		PSetLayouts:        layouts,
	}
	sets := make([]vk.DescriptorSet, cnt)
	err := vk.Error(vk.AllocateDescriptorSets(dp.device, &allocInfo, &(sets[0])))
	if err != nil {
		log.Panicf("Failed to allocate descriptor sets")
	}
	return sets
}

func (dp *DescriptorProvisioner) createDescriptorSetLayout() {
	uboLayoutBinding := vk.DescriptorSetLayoutBinding{
		Binding:            0,                              // <- binding index in vert shader
		DescriptorType:     vk.DescriptorTypeUniformBuffer, // <- type of binding in vert shader
		DescriptorCount:    1,
		StageFlags:         vk.ShaderStageFlags(vk.ShaderStageVertexBit),
		PImmutableSamplers: nil,
	}
	textureSamplerLayoutBinding := vk.DescriptorSetLayoutBinding{
		Binding:            1,                                     // <- binding index in frag shader
		DescriptorType:     vk.DescriptorTypeCombinedImageSampler, // <- type of binding in frag shader
		DescriptorCount:    1,
		StageFlags:         vk.ShaderStageFlags(vk.ShaderStageFragmentBit),
		PImmutableSamplers: nil,
	}
	layoutInfo := vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		PNext:        nil,
		Flags:        0,
		BindingCount: 2,
		PBindings:    []vk.DescriptorSetLayoutBinding{uboLayoutBinding, textureSamplerLayoutBinding},
	}
	dsl, err := com.VKCreateDescriptorSetLayout(dp.device, &layoutInfo, nil)
	if err != nil {
		log.Panicf("Failed to create descriptor set layout")
	}
	dp.descriptorSetLayout = dsl
}

func (dp *DescriptorProvisioner) createModelDescriptorSetLayout() {
	ctxUboLayoutBinding := vk.DescriptorSetLayoutBinding{
		Binding:            0,                              // <- binding index in vert shader
		DescriptorType:     vk.DescriptorTypeUniformBuffer, // <- type of binding in vert shader
		DescriptorCount:    1,
		StageFlags:         vk.ShaderStageFlags(vk.ShaderStageVertexBit),
		PImmutableSamplers: nil,
	}
	layoutInfo := vk.DescriptorSetLayoutCreateInfo{
		SType:        vk.StructureTypeDescriptorSetLayoutCreateInfo,
		PNext:        nil,
		Flags:        0,
		BindingCount: 1,
		PBindings:    []vk.DescriptorSetLayoutBinding{ctxUboLayoutBinding},
	}
	dsl, err := com.VKCreateDescriptorSetLayout(dp.device, &layoutInfo, nil)
	if err != nil {
		log.Panicf("Failed to create descriptor set layout")
	}
	dp.modelDescriptorSetLayout = dsl
}

func (dp *DescriptorProvisioner) createDescriptorPool() {
	uboPoolSize := vk.DescriptorPoolSize{
		Type:            vk.DescriptorTypeUniformBuffer,
		DescriptorCount: MAX_FRAMES_IN_FLIGHT,
	}
	texSamplerPoolSize := vk.DescriptorPoolSize{
		Type:            vk.DescriptorTypeCombinedImageSampler,
		DescriptorCount: MAX_FRAMES_IN_FLIGHT,
	}
	poolInfo := vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		PNext:         nil,
		Flags:         0,
		MaxSets:       MAX_FRAMES_IN_FLIGHT,
		PoolSizeCount: 2,
		PPoolSizes:    []vk.DescriptorPoolSize{uboPoolSize, texSamplerPoolSize},
	}
	var descp vk.DescriptorPool
	if vk.CreateDescriptorPool(dp.device, &poolInfo, nil, &descp) != vk.Success {
		log.Panicf("Failed to create descriptor pool")
	}
	dp.descriptorPool = descp
}

func (dp *DescriptorProvisioner) createModelDescriptorPool() {
	// this should be dynamic somehow
	modelCount := uint32(4)
	uboPoolSize := vk.DescriptorPoolSize{
		Type:            vk.DescriptorTypeUniformBuffer,
		DescriptorCount: 1,
	}
	poolInfo := vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		PNext:         nil,
		Flags:         0,
		MaxSets:       modelCount,
		PoolSizeCount: 1,
		PPoolSizes:    []vk.DescriptorPoolSize{uboPoolSize},
	}
	var descp vk.DescriptorPool
	if vk.CreateDescriptorPool(dp.device, &poolInfo, nil, &descp) != vk.Success {
		log.Panicf("Failed to create descriptor pool")
	}
	dp.modelDescriptorPool = descp
}

func (dp *DescriptorProvisioner) createDescriptorSets(ubos []vk.Buffer, textureSampler vk.Sampler, textureImageView vk.ImageView) {

	layouts := []vk.DescriptorSetLayout{dp.descriptorSetLayout, dp.descriptorSetLayout, dp.descriptorSetLayout}
	dp.descriptorSets = dp.allocDescriptorSets(dp.descriptorPool, layouts)

	for i := 0; i < MAX_FRAMES_IN_FLIGHT; i++ {
		// ubo
		bufferInfo := vk.DescriptorBufferInfo{
			Buffer: ubos[i],
			Offset: 0,
			Range:  model.SizeOfUbo(),
		}
		uboDescriptorWrite := vk.WriteDescriptorSet{
			SType:            vk.StructureTypeWriteDescriptorSet,
			PNext:            nil,
			DstSet:           dp.descriptorSets[i],
			DstBinding:       0,
			DstArrayElement:  0,
			DescriptorCount:  1,
			DescriptorType:   vk.DescriptorTypeUniformBuffer,
			PImageInfo:       nil,
			PBufferInfo:      []vk.DescriptorBufferInfo{bufferInfo},
			PTexelBufferView: nil,
		}

		// textureSampler
		texSampler := vk.DescriptorImageInfo{
			Sampler:     textureSampler,
			ImageView:   textureImageView,
			ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal,
		}
		texSamplerDescriptorWrite := vk.WriteDescriptorSet{
			SType:           vk.StructureTypeWriteDescriptorSet,
			PNext:           nil,
			DstSet:          dp.descriptorSets[i],
			DstBinding:      1, // <-- shader binding location, corresponds to 'layout(binding = 1) uniform sampler2D texSampler;'
			DstArrayElement: 0, // <-- when binding a single texture, this will just be 0 for now. Its the starting index in the binding.
			// assuming I would push 4 texture samplers I could select where they are placed in the array of the binding
			// e.g.: 'layout(binding = 1) uniform sampler2D texSampler[4];' -> pushing 2 samplers and setting it to 2
			// would fill index 2 and 3
			DescriptorCount:  1,
			DescriptorType:   vk.DescriptorTypeCombinedImageSampler,
			PImageInfo:       []vk.DescriptorImageInfo{texSampler},
			PBufferInfo:      nil,
			PTexelBufferView: nil,
		}
		writes := []vk.WriteDescriptorSet{uboDescriptorWrite, texSamplerDescriptorWrite}
		vk.UpdateDescriptorSets(dp.device, uint32(len(writes)), writes, 0, nil)
	}
}

func (dp *DescriptorProvisioner) createModelDescriptorSets(ctxUbos []vk.Buffer) {
	// this holds descriptor sets for 3 models, this needs to be dynamic somehow
	modelCount := uint32(4)
	layouts := []vk.DescriptorSetLayout{dp.modelDescriptorSetLayout, dp.modelDescriptorSetLayout, dp.modelDescriptorSetLayout, dp.modelDescriptorSetLayout}
	allocInfo := vk.DescriptorSetAllocateInfo{
		SType:              vk.StructureTypeDescriptorSetAllocateInfo,
		PNext:              nil,
		DescriptorPool:     dp.modelDescriptorPool,
		DescriptorSetCount: modelCount,
		PSetLayouts:        layouts,
	}
	sets := make([]vk.DescriptorSet, modelCount)
	err := vk.Error(vk.AllocateDescriptorSets(dp.device, &allocInfo, &(sets[0])))
	if err != nil {
		log.Panicf("Failed to allocate descriptor set: %v", err)
	}
	log.Printf("%v", sets)
	dp.modelDescriptorSets = sets

	for i := 0; i < int(modelCount); i++ {
		// ctxubo
		ctxBufferInfo := vk.DescriptorBufferInfo{
			Buffer: ctxUbos[i],
			Offset: 0,
			Range:  model.SizeOfCtxUbo(),
		}
		ctxUboDescriptorWrite := vk.WriteDescriptorSet{
			SType:            vk.StructureTypeWriteDescriptorSet,
			PNext:            nil,
			DstSet:           dp.modelDescriptorSets[i],
			DstBinding:       0,
			DstArrayElement:  0,
			DescriptorCount:  1,
			DescriptorType:   vk.DescriptorTypeUniformBuffer,
			PImageInfo:       nil,
			PBufferInfo:      []vk.DescriptorBufferInfo{ctxBufferInfo},
			PTexelBufferView: nil,
		}
		writes := []vk.WriteDescriptorSet{ctxUboDescriptorWrite}
		vk.UpdateDescriptorSets(dp.device, uint32(len(writes)), writes, 0, nil)
	}
}
