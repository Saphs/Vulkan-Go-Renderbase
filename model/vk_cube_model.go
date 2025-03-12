package model

import vm "local/vector_math"

func NewCubeModel(name string) *Model {

	// Expected size in memory -> 64 Byte with 4 Bytes of padding as we have 8 Byte words on a 64Bit machine
	v := []Vertex{ // 24 * 8 = 192 Byte
		{ // 12 + 12 + 8 = 32 Byte [0]
			Pos:      vm.Vec3{X: -0.5, Y: -0.5, Z: -0.5}, // 12 Byte (float32 * 3, no padding)
			Color:    vm.Vec3{X: 1, Y: 0, Z: 0},          // 12 Byte (float32 * 3, no padding)
			TexCoord: vm.Vec2{X: 1, Y: 1},                // 8 Byte (float32 * 2, no padding)
		},
		{ // [1]
			Pos:      vm.Vec3{X: 0.5, Y: -0.5, Z: -0.5},
			Color:    vm.Vec3{X: 0, Y: 1, Z: 0},
			TexCoord: vm.Vec2{X: 0, Y: 1},
		},
		{ // [2]
			Pos:      vm.Vec3{X: 0.5, Y: 0.5, Z: -0.5},
			Color:    vm.Vec3{X: 0, Y: 0, Z: 1},
			TexCoord: vm.Vec2{X: 0, Y: 0},
		},
		{ // [3]
			Pos:      vm.Vec3{X: -0.5, Y: 0.5, Z: -0.5},
			Color:    vm.Vec3{X: 1, Y: 0.5, Z: 1},
			TexCoord: vm.Vec2{X: 1, Y: 0},
		},
		{ // 8 + 12 = 20 Byte [4]
			Pos:      vm.Vec3{X: -0.5, Y: -0.5, Z: 0.5}, // 12 Byte (float32 * 3, no padding)
			Color:    vm.Vec3{X: 1, Y: 0.5, Z: 0.5},     // 12 Byte (float32 * 3, no padding)
			TexCoord: vm.Vec2{X: 1, Y: 1},
		},
		{ // [5]
			Pos:      vm.Vec3{X: 0.5, Y: -0.5, Z: 0.5},
			Color:    vm.Vec3{X: 0.5, Y: 1, Z: 0.5},
			TexCoord: vm.Vec2{X: 0, Y: 1},
		},
		{ // [6]
			Pos:      vm.Vec3{X: 0.5, Y: 0.5, Z: 0.5},
			Color:    vm.Vec3{X: 0.5, Y: 0.5, Z: 1},
			TexCoord: vm.Vec2{X: 0, Y: 0},
		},
		{ // [7]
			Pos:      vm.Vec3{X: -0.5, Y: 0.5, Z: 0.5},
			Color:    vm.Vec3{X: 0, Y: 0.5, Z: 0},
			TexCoord: vm.Vec2{X: 1, Y: 0},
		},
	}

	id := []uint32{
		2, 1, 0, 0, 3, 2, // front
		5, 1, 6, 1, 2, 6, // right
		4, 5, 6, 7, 4, 6, // back
		4, 7, 0, 0, 7, 3, // left
		0, 1, 5, 5, 4, 0, // top
		3, 7, 6, 2, 3, 6, // bottom
	}

	mesh := NewMesh(v, id)
	return NewModel(mesh, name)
}
