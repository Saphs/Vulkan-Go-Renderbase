package vector_math

import "math"

// ToRad is a helper function to turn radians to degree
func ToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// ToDeg is a helper function to turn degree to radians
func ToDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}

// Apply is a hacky way to multiply a vec3 by a Mat4x4 by using a homogeneous
// coordinate that can be specified.
// ToDo: Replace this with a generalized Vec.Mul() function or similar
func Apply(v Vec3, w float32, m Mat) Vec3 {
	v4 := []float32{v.X, v.Y, v.Z, w}
	return Vec3{
		(v4[0] * m[0][0]) + (v4[1] * m[0][1]) + (v4[2] * m[0][2]) + (v4[3] * m[0][3]),
		(v4[0] * m[1][0]) + (v4[1] * m[1][1]) + (v4[2] * m[1][2]) + (v4[3] * m[1][3]),
		(v4[0] * m[2][0]) + (v4[1] * m[2][1]) + (v4[2] * m[2][2]) + (v4[3] * m[2][3]),
	}
}
