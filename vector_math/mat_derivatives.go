package vector_math

import (
	"math"
)

func New4x4RotXMat(rad float64) Mat {
	m, _ := NewMat(4, 4)
	m[0][0] = 1
	m[1][1] = float32(math.Cos(rad))
	m[1][2] = -float32(math.Sin(rad))
	m[2][1] = float32(math.Sin(rad))
	m[2][2] = float32(math.Cos(rad))
	m[3][3] = 1
	return m
}

func New4x4RotYMat(rad float64) Mat {
	m, _ := NewMat(4, 4)
	m[0][0] = float32(math.Cos(rad))
	m[0][2] = float32(math.Sin(rad))
	m[1][1] = 1
	m[2][0] = -float32(math.Sin(rad))
	m[2][2] = float32(math.Cos(rad))
	m[3][3] = 1
	return m
}

func New4x4RotZMat(rad float64) Mat {
	m, _ := NewMat(4, 4)
	m[0][0] = float32(math.Cos(rad))
	m[0][1] = -float32(math.Sin(rad))
	m[1][0] = float32(math.Sin(rad))
	m[1][1] = float32(math.Cos(rad))
	m[2][2] = 1
	m[3][3] = 1
	return m
}

func New4x4RotMat(yaw float64, pitch float64, roll float64) Mat {
	mx := New4x4RotXMat(roll)
	my := New4x4RotYMat(pitch)
	mz := New4x4RotZMat(yaw)
	mzy, _ := mz.Mult(&my)
	rot, _ := mzy.Mult(&mx)
	return rot
}

func NewUnitMat(s uint) Mat {
	um, _ := NewMat(s, s)
	for i := range um {
		um[i][i] = 1
	}
	return um
}

func NewRotation(rad float64, axis Vec3) Mat {
	ux := axis.X
	uy := axis.Y
	uz := axis.Z
	if (ux*ux)+(uy*uy)+(uz*uz) != 1 {
		norm := axis.Norm()
		ux = norm.X
		uy = norm.Y
		uz = norm.Z
	}
	cosT := float32(math.Cos(rad))
	sinT := float32(math.Sin(rad))
	rm := NewUnitMat(4)
	rm[0][0] = cosT + ((ux * ux) * (1 - cosT))
	rm[0][1] = (ux*uy)*(1-cosT) - (uz * sinT)
	rm[0][2] = (ux*uz)*(1-cosT) + (uy * sinT)

	rm[1][0] = (uy*ux)*(1-cosT) + (uz * sinT)
	rm[1][1] = cosT + (uy*uy)*(1-cosT)
	rm[1][2] = (uy*uz)*(1-cosT) - (ux * sinT)

	rm[2][0] = (uz*ux)*(1-cosT) - (uy * sinT)
	rm[2][1] = (uz*uy)*(1-cosT) + (ux * sinT)
	rm[2][2] = cosT + (uz*uz)*(1-cosT)

	return rm
}

// NewPerspectiveProjection implemented after: https://www.youtube.com/watch?v=U0_ONQQ5ZNM
func NewPerspectiveOld(fovy float64, aspect float64, near float32, far float32) Mat {
	focalLen := 1 / math.Tan(fovy/2)
	m, _ := NewMat(4, 4)
	m[0][0] = float32(focalLen / aspect)
	m[1][1] = float32(focalLen)
	m[2][2] = far / (far - near)
	m[2][3] = 1
	m[3][2] = -(far * near) / (far - near)
	return m
}

func NewPerspective(n float32, f float32) Mat {
	m, _ := NewMat(4, 4)
	m[0][0] = n
	m[1][1] = n
	m[2][2] = f + n
	m[2][3] = -f * n
	m[3][2] = 1
	return m
}

// NewOrthographicProjection constructs a new matrix representing an orthographic projection from
// a cuboid on to Vulkan's canonical view volume (CVV), which spans from (-1, 1, 0) to (1, -1, 1). The
// returned projection takes any cuboid spanning from lbn (Left-Bottom-Near) to rtf (Right-Top-Far)
// and moves its values into the CVV, which is in turn displayed.
// -------------------------------------------------------------
// Setting the orthographic view volume to have the same aspect ratio as the viewport will avoid stretching
// any points. To do this, let the following term be true: "right - left = aspect * (bottom - top)". The
// current aspect ratio of the viewport can be retrieved via the swap chain's width and height.
func NewOrthographicProjection(lbn Vec3, rtf Vec3) Mat {
	// Scaling factors assume the given CVV cuboids dimensions as fixed (width: 2, height: 2, depth: 2)
	mScale := NewScale(Vec3{
		X: float32(2 / (math.Abs(float64(rtf.X) - float64(lbn.X)))),
		Y: float32(2 / (math.Abs(float64(lbn.Y) - float64(rtf.Y)))),
		Z: float32(1 / (math.Abs(float64(rtf.Z) - float64(lbn.Z)))),
	})
	mTrans := NewTranslation(Vec3{
		X: -(rtf.X + lbn.X) / (rtf.X - lbn.X),
		Y: -(lbn.X + rtf.X) / (lbn.X - rtf.X),
		Z: -lbn.Z / (rtf.Z - lbn.Z),
	})
	mOrt, _ := mScale.Mult(&mTrans)
	return mOrt
}

// NewLookAt implemented after http://www.opengl.org/sdk/docs/man2/xhtml/gluLookAt.xml
func NewLookAt(camPos Vec3, camTarget Vec3, up Vec3) Mat {
	camDir := camPos.Sub(camTarget).Norm()
	camRight := up.Cross(camDir).Norm()
	camUp := camDir.Cross(camRight)

	m, _ := NewMat(4, 4)

	m[0][0] = camRight.X
	m[0][1] = camRight.Y
	m[0][2] = camRight.Z

	m[1][0] = camUp.X
	m[1][1] = camUp.Y
	m[1][2] = camUp.Z

	m[2][0] = camDir.X
	m[2][1] = camDir.Y
	m[2][2] = camDir.Z

	m[3][3] = 1

	m = m.Transpose()

	mP := NewTranslation(Vec3{
		X: -camPos.X,
		Y: -camPos.Y,
		Z: -camPos.Z,
	})

	m, _ = mP.Mult(&m)
	return m
}

func NewScale(s Vec3) Mat {
	sm := NewUnitMat(4)
	sm[0][0] = s.X
	sm[1][1] = s.Y
	sm[2][2] = s.Z
	return sm
}

func NewTranslation(t Vec3) Mat {
	tm := NewUnitMat(4)
	tm[0][3] = t.X
	tm[1][3] = t.Y
	tm[2][3] = t.Z
	return tm
}

func NewTranslationT(t Vec3) Mat {
	tm := NewUnitMat(4)
	tm[3][0] = t.X
	tm[3][1] = t.Y
	tm[3][2] = t.Z
	return tm
}
