#version 450

//ubos
layout(set = 0, binding = 0) uniform UniformBufferObject {
    mat4 view;
    mat4 proj;
} ubo;

layout(set = 1, binding = 0) uniform ModelUniformBufferObject {
    int modelType;
} ctx;


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
    gl_Position = ubo.proj * ubo.view * pc.model * vec4(inPosition, 1.0);
    fragColor = inColor;
    vec2 tex = inTexColor;
    if (ctx.modelType == 0) {
        //gl_Position = vec4(inPosition, 1.0);
        tex = vec2(0.0, 0.0);
    }

    fragTexColor = tex;
}