package vector_math

import "math"

type Vec2 struct {
	X, Y float32
}

func (v Vec2) Dot(w Vec2) float32 {
	return (v.X * w.X) + (v.Y * w.Y)
}

func (v Vec2) Sub(w Vec2) Vec2 {
	return Vec2{
		X: v.X - w.X,
		Y: v.Y - w.Y,
	}
}

func (v Vec2) Add(w Vec2) Vec2 {
	return Vec2{
		X: v.X + w.X,
		Y: v.Y + w.Y,
	}
}

func (v Vec2) ScalarMul(factor float32) Vec2 {
	return Vec2{
		X: v.X * factor,
		Y: v.Y * factor,
	}
}

func (v Vec2) len() float32 {
	return float32(math.Sqrt(float64((v.X * v.X) + (v.Y * v.Y))))
}

func (v Vec2) Norm() Vec2 {
	l := v.len()
	return Vec2{
		X: v.X / l,
		Y: v.Y / l,
	}
}
