package vector_math

type Mesh struct {
	Vertices []Vertex
	VIndices []uint32
	ModelMat Mat
}

func NewMesh(v []Vertex, id []uint32) *Mesh {
	return &Mesh{
		Vertices: v,
		VIndices: id,
		ModelMat: NewUnitMat(4),
	}
}
