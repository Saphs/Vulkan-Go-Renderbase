package vector_math

import "math"

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

// NewPerspective implemented after: https://registry.khronos.org/OpenGL-Refpages/gl2.1/xhtml/gluPerspective.xml
func NewPerspective(fovy float64, aspect float64, zNear float32, zFar float32) Mat {
	f := math.Tan(fovy / 2)
	m := NewUnitMat(4)
	m[0][0] = float32(1 / (aspect * f))
	m[1][1] = float32(1 / f)
	m[2][2] = zFar / (zFar - zNear)
	m[2][3] = 1
	m[3][2] = -(zFar * zNear) / (zFar - zNear)
	return m
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
