module GPU_fluid_simulation

go 1.22

toolchain go1.22.2

require (
	github.com/goki/vulkan v1.0.7
	github.com/veandco/go-sdl2 v0.4.38
	local/vector_math v0.0.0-00010101000000-000000000000
)

replace local/vector_math => ./vector_math
