package vector_math

import "math"

type Vec3 struct {
	X, Y, Z float32
}

func (v Vec3) Cross(w Vec3) Vec3 {
	return Vec3{
		X: (v.Y * w.Z) - (v.Z * w.Y),
		Y: (v.Z * w.X) - (v.X * w.Z),
		Z: (v.X * w.Y) - (v.Y * w.X),
	}
}

func (v Vec3) Sub(w Vec3) Vec3 {
	return Vec3{
		X: v.X - w.X,
		Y: v.Y - w.Y,
		Z: v.Z - w.Z,
	}
}

func (v Vec3) Norm() Vec3 {
	l := float32(math.Sqrt(float64((v.X * v.X) + (v.Y * v.Y) + (v.Z * v.Z))))
	return Vec3{
		X: v.X / l,
		Y: v.Y / l,
		Z: v.Z / l,
	}
}
