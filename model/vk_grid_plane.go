package model

import vm "local/vector_math"

func NewGridPlane(name string) *Model {

	// Expected size in memory -> 64 Byte with 4 Bytes of padding as we have 8 Byte words on a 64Bit machine
	v := []Vertex{ // 24 * 8 = 192 Byte
		{ // 12 + 12 + 8 = 32 Byte [0]
			Pos:      vm.Vec3{X: -1, Y: -1, Z: 0}, // 12 Byte (float32 * 3, no padding)
			Color:    vm.Vec3{X: 1, Y: 0, Z: 0},   // 12 Byte (float32 * 3, no padding)
			TexCoord: vm.Vec2{X: 0, Y: 0},         // 8 Byte (float32 * 2, no padding)
		},
		{ // [1]
			Pos:      vm.Vec3{X: -1, Y: 1, Z: 0},
			Color:    vm.Vec3{X: 0, Y: 1, Z: 0},
			TexCoord: vm.Vec2{X: 0, Y: 1},
		},
		{ // [2]
			Pos:      vm.Vec3{X: 1, Y: 1, Z: 0},
			Color:    vm.Vec3{X: 0, Y: 0, Z: 1},
			TexCoord: vm.Vec2{X: 1, Y: 1},
		},
		{ // [3]
			Pos:      vm.Vec3{X: 1, Y: -1, Z: 0},
			Color:    vm.Vec3{X: 1, Y: 0.5, Z: 1},
			TexCoord: vm.Vec2{X: 1, Y: 0},
		},
	}

	id := []uint32{
		0, 1, 2,
		2, 3, 0,
	}

	mesh := NewMesh(v, id)
	return NewModel(mesh, name)
}
