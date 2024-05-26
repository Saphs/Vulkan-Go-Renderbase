package main

import "local/vector_math"

type Mesh struct {
	Vertices []Vertex
	VIndices []uint32
	ModelMat vector_math.Mat
}

func NewMesh(v []Vertex, id []uint32) *Mesh {
	return &Mesh{
		Vertices: v,
		VIndices: id,
		ModelMat: vector_math.NewUnitMat(4),
	}
}
