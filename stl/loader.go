package stl

import (
	"GPU_fluid_simulation/model"
	"encoding/binary"
	"local/vector_math"
	"log"
	"math"
	"os"
)

func ReadStlFile(path string) *model.Mesh {
	log.Printf("Reading stl file %s", path)
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	header := b[:80]
	tCntBits := binary.LittleEndian.Uint32(b[80:84])
	byteCnt := len(b[84:]) / 1024
	log.Printf("Successfully read stl file, Header: '%s', Triangle Count: %d, Triangle memory size: %d KiB", header, tCntBits, byteCnt)
	// log.Printf("Mesh: %v", toMesh(b[84:], tCntBits))
	return toMesh(b[84:], tCntBits)
}

func toMesh(bytes []byte, triangleCnt uint32) *model.Mesh {
	stride := 50
	v := make([]model.Vertex, triangleCnt*3)
	vi := 0

	id := make([]uint32, triangleCnt*3)
	idxi := uint32(0)

	for i := 0; i < len(bytes); i += stride {
		normal := toVec3(bytes[i : i+12])
		v1 := toVec3(bytes[i+12 : i+24])
		v[vi] = model.Vertex{
			Pos:   v1,
			Color: normal,
		}
		vi++
		id[idxi] = idxi
		idxi++

		v2 := toVec3(bytes[i+24 : i+36])
		v[vi] = model.Vertex{
			Pos:   v2,
			Color: normal,
		}
		vi++
		id[idxi] = idxi
		idxi++

		v3 := toVec3(bytes[i+36 : i+48])
		v[vi] = model.Vertex{
			Pos:   v3,
			Color: normal,
		}
		vi++
		id[idxi] = idxi
		idxi++

		// attr := bytes[i+48 : i+50]
		// log.Printf("normal: %v, v1: %v, v2: %v, v3: %v, attr: %v", normal, v1, v2, v3, attr)
	}

	return model.NewMesh(v, id)
}

func toVec3(bytes []byte) vector_math.Vec3 {
	return vector_math.Vec3{
		X: toFloat32(bytes[:4]),
		Y: toFloat32(bytes[4:8]),
		Z: toFloat32(bytes[8:12]),
	}
}

func toFloat32(bytes []byte) float32 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}
