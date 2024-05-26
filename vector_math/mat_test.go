package vector_math

import (
	"testing"
)

// TestNewMat calls NewMat and confirms some general size constraints
func TestNewMat(t *testing.T) {

	mat0, err := NewMat(0, 0)
	if mat0 != nil {
		t.Errorf("Should not be able to create mat0: %s", mat0.ToString())
	}
	mat1, err := NewMat(1, 1)
	if err != nil {
		t.Errorf("Error creating matrix of size 1x1: %s", err)
	}
	mat2, err := NewMat(2, 2)
	if err != nil {
		t.Errorf("Error creating matrix of size 2x2: %s", err)
	}
	mat3, err := NewMat(3, 3)
	if err != nil {
		t.Errorf("Error creating matrix of size 3x3: %s", err)
	}
	mat4, err := NewMat(4, 4)
	if err != nil {
		t.Errorf("Error creating matrix of size 4x4: %s", err)
	}
	mat5, err := NewMat(5, 5)
	if err != nil {
		t.Errorf("Error creating matrix of size 5x5: %s", err)
	}
	mat6, err := NewMat(6, 6)
	if err != nil {
		t.Errorf("Error creating matrix of size 6x6: %s", err)
	}

	if mat1.ByteSize() != 4 {
		t.Errorf("mat1 should have byte size: %d", 4)
	}
	if mat2.ByteSize() != 16 {
		t.Errorf("mat2 should have byte size: %d", 16)
	}
	if mat3.ByteSize() != 36 {
		t.Errorf("mat3 should have byte size: %d", 36)
	}
	if mat4.ByteSize() != 64 {
		t.Errorf("mat4 should have byte size: %d", 64)
	}
	if mat5.ByteSize() != 100 {
		t.Errorf("mat5 should have byte size: %d", 100)
	}
	if mat6.ByteSize() != 144 {
		t.Errorf("mat6 should have byte size: %d but was %d", 144, mat6.ByteSize())
	}

}

func TestRotationX(t *testing.T) {
	t.Logf("RotationX:")
	mrx := New4x4RotXMat(ToRad(90))
	mrxComplex := NewRotation(ToRad(90), Vec3{X: 1})

	if !mrx.Equals(&mrxComplex) {
		t.Errorf(
			"RotX not equal to generic roation around X. RotX: \n%s\n Rotation around x-axis: \n%s",
			mrx.ToString(),
			mrxComplex.ToString(),
		)
	}
}

func TestRotationY(t *testing.T) {
	t.Logf("RotationY:")
	mry := New4x4RotYMat(ToRad(90))
	mryComplex := NewRotation(ToRad(90), Vec3{Y: 1})

	if !mry.Equals(&mryComplex) {
		t.Errorf(
			"RotY not equal to generic roation around Y. RotY: \n%s\n Rotation around y-axis: \n%s",
			mry.ToString(),
			mryComplex.ToString(),
		)
	}
}

func TestRotationZ(t *testing.T) {
	t.Logf("RotationZ:")
	mrz := New4x4RotZMat(ToRad(90))
	mrzComplex := NewRotation(ToRad(90), Vec3{Z: 1})

	if !mrz.Equals(&mrzComplex) {
		t.Errorf(
			"RotZ not equal to generic roation around Z. RotZ: \n%s\n Rotation around z-axis: \n%s",
			mrz.ToString(),
			mrzComplex.ToString(),
		)
	}
}

func TestArbitraryRotation(t *testing.T) {
	t.Logf("Rotation arbitrary:")
	mr := NewRotation(ToRad(-74), Vec3{X: -0.5, Y: 1, Z: 1})
	mrExample := NewUnitMat(4)
	mrExample[0][0] = 0.3561221
	mrExample[0][1] = 0.47987163
	mrExample[0][2] = -0.8018106

	mrExample[1][0] = -0.8018106
	mrExample[1][1] = 0.5975763
	mrExample[1][2] = 0.0015183985

	mrExample[2][0] = 0.47987163
	mrExample[2][1] = 0.6423595
	mrExample[2][2] = 0.5975763

	if !mr.Equals(&mrExample) {
		t.Errorf(
			"Arbitrary rotation didnt match expectations. expectation: \n%s\n actual: \n%s",
			mr.ToString(),
			mrExample.ToString(),
		)
	}
}

func TestNewPerspective(t *testing.T) {

}

func TestUnroll(t *testing.T) {
	t.Logf("Rotation arbitrary:")
	m, _ := NewMat(4, 4)
	m.FillRng(0, 10)
	t.Logf("%s", m.Describe())
	t.Logf("%v", m.Unroll())
}

func TestTranspose(t *testing.T) {
	m, _ := NewMat(3, 4)
	m.FillRng(0, 10)
	mT := m.Transpose()
	t.Logf("%s", m.Describe())
	t.Logf("%s", mT.Describe())
}
