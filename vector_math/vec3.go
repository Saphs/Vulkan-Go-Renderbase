package vector_math

import (
	"math"
)

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

func (v Vec3) Dot(w Vec3) float32 {
	return (v.X * w.X) + (v.Y * w.Y) + (v.Z * w.Z)
}

func (v Vec3) Sub(w Vec3) Vec3 {
	return Vec3{
		X: v.X - w.X,
		Y: v.Y - w.Y,
		Z: v.Z - w.Z,
	}
}

func (v Vec3) Add(w Vec3) Vec3 {
	return Vec3{
		X: v.X + w.X,
		Y: v.Y + w.Y,
		Z: v.Z + w.Z,
	}
}

func (v Vec3) ScalarMul(factor float32) Vec3 {
	return Vec3{
		X: v.X * factor,
		Y: v.Y * factor,
		Z: v.Z * factor,
	}
}

func (v Vec3) len() float32 {
	return float32(math.Sqrt(float64((v.X * v.X) + (v.Y * v.Y) + (v.Z * v.Z))))
}

func (v Vec3) Norm() Vec3 {
	l := v.len()
	return Vec3{
		X: v.X / l,
		Y: v.Y / l,
		Z: v.Z / l,
	}
}
