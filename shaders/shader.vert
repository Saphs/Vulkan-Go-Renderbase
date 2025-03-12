#version 450

//ubos
layout(binding = 0) uniform UniformBufferObject {
    mat4 view;
    mat4 proj;
} ubo;

//push constants
layout( push_constant ) uniform constants {
    mat4 model;
} pc;

layout(location = 0) in vec3 inPosition;
layout(location = 1) in vec3 inColor;
layout(location = 2) in vec2 inTexColor;

layout(location = 0) out vec3 fragColor;
layout(location = 1) out vec2 fragTexColor;

void main() {
    gl_Position = vec4(inPosition, 1.0) * pc.model * ubo.view * ubo.proj;
    fragColor = inColor;
    fragTexColor = inTexColor;
}