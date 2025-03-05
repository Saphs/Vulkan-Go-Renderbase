package model

import (
	"local/vector_math"
	"log"
	"math"
)

const (
	CAM_PERSPECTIVE_PROJECTION  = iota
	CAM_ORTHOGRAPHIC_PROJECTION = iota
)

type Camera struct {
	ProjectionType int

	Fov    float32
	Aspect float32
	Near   float32
	Far    float32

	Pos        vector_math.Vec3
	LookDir    vector_math.Vec3
	LookTarget *vector_math.Vec3
	Up         vector_math.Vec3

	View vector_math.Mat
}

func NewCamera(fov float32, near float32, far float32) *Camera {
	return &Camera{
		Fov:        fov,
		Near:       near,
		Far:        far,
		LookDir:    vector_math.Vec3{Z: 1},
		LookTarget: nil,
		Up:         vector_math.Vec3{Y: -1},
		View:       vector_math.NewUnitMat(4),
	}
}

func (c *Camera) Move(v vector_math.Vec3) {
	c.Pos = c.Pos.Add(v)
}

func (c *Camera) Turn(deg float64, axis vector_math.Vec3) {
	rm := vector_math.NewRotation(vector_math.ToRad(deg), axis)
	c.LookDir = vector_math.Apply(c.LookDir, 0, rm)
}

func (c *Camera) SetTarget(v vector_math.Vec3) {
	c.LookTarget = &v
}

func (c *Camera) GetProjection() vector_math.Mat {
	switch c.ProjectionType {
	case CAM_PERSPECTIVE_PROJECTION:
		return newPerspectiveProjection(
			vector_math.ToRad(float64(c.Fov)), float64(c.Aspect), c.Near, c.Far,
		)
	case CAM_ORTHOGRAPHIC_PROJECTION:
		return newOrthographicProjection(
			vector_math.Vec3{X: -c.Aspect, Y: 1, Z: c.Near}, vector_math.Vec3{X: c.Aspect, Y: -1, Z: c.Far},
		)
	default:
		log.Printf("Failed to select projection type, returning identity.")
		return vector_math.NewUnitMat(4)
	}
}

func (c *Camera) GetView() vector_math.Mat {
	if c.LookTarget != nil {
		return NewTargetView(c.Pos, *c.LookTarget, c.Up)
	} else {
		m := NewDirectionView(c.Pos, c.LookDir, c.Up)
		return m
	}
}

// newPerspectiveProjection implemented after: https://www.youtube.com/watch?v=U0_ONQQ5ZNM
func newPerspectiveProjection(fovy float64, aspect float64, near float32, far float32) vector_math.Mat {
	focalLen := 1 / math.Tan(fovy/2)
	m, _ := vector_math.NewMat(4, 4)
	m[0][0] = float32(focalLen / aspect)
	m[1][1] = float32(focalLen)
	m[2][2] = far / (far - near)
	m[2][3] = -(far * near) / (far - near)
	m[3][2] = 1
	return m
}

// newOrthographicProjection constructs a new matrix representing an orthographic projection from
// a cuboid on to Vulkan's canonical view volume (CVV), which spans from (-1, 1, 0) to (1, -1, 1). The
// returned projection takes any cuboid spanning from lbn (Left-Bottom-Near) to rtf (Right-Top-Far)
// and moves its values into the CVV, which is in turn displayed.
// -------------------------------------------------------------
// Setting the orthographic view volume to have the same aspect ratio as the viewport will avoid stretching
// any points. To do this, let the following term be true: "right - left = aspect * (bottom - top)". The
// current aspect ratio of the viewport can be retrieved via the swap chain's width and height.
func newOrthographicProjection(lbn vector_math.Vec3, rtf vector_math.Vec3) vector_math.Mat {
	// Scaling factors assume the given CVV cuboids dimensions as fixed (width: 2, height: 2, depth: 2)
	mScale := vector_math.NewScale(vector_math.Vec3{
		X: float32(2 / (math.Abs(float64(rtf.X) - float64(lbn.X)))),
		Y: float32(2 / (math.Abs(float64(lbn.Y) - float64(rtf.Y)))),
		Z: float32(1 / (math.Abs(float64(rtf.Z) - float64(lbn.Z)))),
	})
	mTrans := vector_math.NewTranslation(vector_math.Vec3{
		X: -(rtf.X + lbn.X) / (rtf.X - lbn.X),
		Y: -(lbn.X + rtf.X) / (lbn.X - rtf.X),
		Z: -lbn.Z / (rtf.Z - lbn.Z),
	})
	mOrt, _ := mScale.Mult(&mTrans)
	return mOrt
}

func NewDirectionView(pos vector_math.Vec3, dir vector_math.Vec3, up vector_math.Vec3) vector_math.Mat {
	// construct orthonormal basis vectors
	w := dir.Norm()
	u := w.Cross(up).Norm()
	v := w.Cross(u)
	m := vector_math.NewUnitMat(4)
	m[0][0] = u.X
	m[0][1] = u.Y
	m[0][2] = u.Z
	m[1][0] = v.X
	m[1][1] = v.Y
	m[1][2] = v.Z
	m[2][0] = w.X
	m[2][1] = w.Y
	m[2][2] = w.Z
	m[0][3] = -u.Dot(pos)
	m[1][3] = -v.Dot(pos)
	m[2][3] = -w.Dot(pos)
	return m
}

func NewTargetView(pos vector_math.Vec3, target vector_math.Vec3, up vector_math.Vec3) vector_math.Mat {
	d := target.Sub(pos)
	if d.Len() == 0 {
		log.Printf("Failed to calculate view direction, target - position = [0,0,0]. Setting d to z-axis.")
		d = vector_math.Vec3{Z: 1}
	}
	return NewDirectionView(pos, d, up)
}

func NewAngleView(pos vector_math.Vec3, rot vector_math.Vec3) vector_math.Mat {
	c1 := float32(math.Cos(vector_math.ToRad(float64(rot.X))))
	s1 := float32(math.Sin(vector_math.ToRad(float64(rot.X))))
	c2 := float32(math.Cos(vector_math.ToRad(float64(rot.Y))))
	s2 := float32(math.Sin(vector_math.ToRad(float64(rot.Y))))
	c3 := float32(math.Cos(vector_math.ToRad(float64(rot.Z))))
	s3 := float32(math.Sin(vector_math.ToRad(float64(rot.Z))))
	u := vector_math.Vec3{
		X: c2 * c3,
		Y: (s1 * s3) + (c1 * c3 * s2),
		Z: (c3 * s1 * s2) - (c1 * s3),
	}
	v := vector_math.Vec3{
		X: -s2,
		Y: c1 * c2,
		Z: c2 * s1,
	}
	w := vector_math.Vec3{
		X: c2 * s3,
		Y: (c1 * s2 * s3) - (c3 * s1),
		Z: (c1 * c3) + (s1 * s2 * s3),
	}
	m, _ := vector_math.NewMat(4, 4)
	m[0][0] = u.X
	m[0][1] = u.Y
	m[0][2] = u.Z
	m[1][0] = v.X
	m[1][1] = v.Y
	m[1][2] = v.Z
	m[2][0] = w.X
	m[2][1] = w.Y
	m[2][2] = w.Z
	m[3][0] = -u.Dot(pos)
	m[3][1] = -v.Dot(pos)
	m[3][2] = -w.Dot(pos)
	return m
}
