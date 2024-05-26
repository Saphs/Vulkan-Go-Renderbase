package vector_math

import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"strings"
	"unsafe"
)

type Mat [][]float32

func NewMat(r uint, c uint) (Mat, error) {
	if r == 0 || c == 0 {
		return nil, errors.New("cannot construct 0-sized matrix")
	}
	m := make([][]float32, r)
	for i := range m {
		m[i] = make([]float32, c)
	}
	return m, nil
}

func (m *Mat) Add(b *Mat) (Mat, error) {
	rowsA, colsA := (*m).Size()
	rowsB, colsB := (*b).Size()
	if rowsA != rowsB || colsA != colsB {
		msg := fmt.Sprintf(
			"can't add %dx%d matrix with %dx%d matrix, matracies not of equal size",
			rowsA, colsA, rowsB, colsB,
		)
		return nil, errors.New(msg)
	}
	C, _ := NewMat(uint(rowsA), uint(colsA))
	for i := 0; i < rowsA; i++ {
		for j := 0; j < colsA; j++ {
			C[i][j] = (*m)[i][j] + (*b)[i][j]
		}
	}
	return C, nil
}

func (m *Mat) Sub(b *Mat) (Mat, error) {
	rowsA, colsA := (*m).Size()
	rowsB, colsB := (*b).Size()
	if rowsA == rowsB && colsA == colsB {
		msg := fmt.Sprintf(
			"can't add %dx%d matrix with %dx%d matrix, matracies not of equal size",
			rowsA, colsA, rowsB, colsB,
		)
		return nil, errors.New(msg)
	}
	C, _ := NewMat(uint(rowsA), uint(colsA))
	for i := 0; i < rowsA; i++ {
		for j := 0; j < colsA; j++ {
			C[i][j] = (*m)[i][j] - (*b)[i][j]
		}
	}
	return C, nil
}

func (m *Mat) Mult(b *Mat) (Mat, error) {
	rowsA, colsA := (*m).Size()
	rowsB, colsB := (*b).Size()
	if colsA != rowsB {
		msg := fmt.Sprintf(
			"can't multiply %dx%d matrix with %dx%d matrix, size of columns and rows do not match",
			rowsA, colsA, rowsB, colsB,
		)
		return nil, errors.New(msg)
	}
	C, _ := NewMat(uint(rowsA), uint(colsB))
	for i := 0; i < rowsA; i++ {
		for j := 0; j < colsB; j++ {
			for k := 0; k < colsA; k++ {
				C[i][j] += (*m)[i][k] * (*b)[k][j]
			}
		}
	}
	return C, nil
}

func (m *Mat) Transpose() Mat {
	mT, _ := NewMat(uint(m.ColCnt()), uint(m.RowCnt()))
	for i := range *m {
		for j := range (*m)[i] {
			mT[j][i] = (*m)[i][j]
		}
	}
	return mT
}

func (m *Mat) Equals(b *Mat) bool {
	rowsA, colsA := (*m).Size()
	rowsB, colsB := (*b).Size()
	if rowsA != rowsB || colsA != colsB {
		return false
	}
	for i := 0; i < rowsA; i++ {
		for j := 0; j < colsA; j++ {
			if (*m)[i][j] != (*b)[i][j] {
				log.Printf("(*m)[i][j] != (*b)[i][j] -> %f != %f", (*m)[i][j], (*b)[i][j])
				return false
			}
		}
	}
	return true
}

// Helper functions

func rng(min, max int) int {
	return rand.IntN(max-min) + min
}

func (m *Mat) Rotate(rad float64, axis Vec3) (Mat, error) {
	rm := NewRotation(rad, axis)
	rm.Transpose()
	res, err := m.Mult(&rm)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *Mat) Translate(move Vec3) (Mat, error) {
	tm := NewTranslation(move)
	res, err := m.Mult(&tm)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *Mat) Scale(factors Vec3) (Mat, error) {
	sm := NewScale(factors)
	res, err := m.Mult(&sm)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *Mat) Fill(f float32) {
	for i := range *m {
		for j := range (*m)[i] {
			(*m)[i][j] = f
		}
	}
}

func (m *Mat) FillRng(min float32, max float32) {
	for i := range *m {
		for j := range (*m)[i] {
			(*m)[i][j] = float32(rng(int(min), int(max)))
		}
	}
}

// Description functions

func (m *Mat) RowCnt() int {
	return len(*m)
}

func (m *Mat) ColCnt() int {
	return len((*m)[0])
}

func (m *Mat) Size() (int, int) {
	return (*m).RowCnt(), (*m).ColCnt()
}

func (m *Mat) ByteSize() int {
	return int(unsafe.Sizeof((*m)[0][0])) * (*m).RowCnt() * (*m).ColCnt()
}

func (m *Mat) Unroll() []float32 {
	f := make([]float32, m.RowCnt()*m.ColCnt())
	for i := range f {
		f[i] = (*m)[int(math.Floor(float64(i/m.RowCnt())))][i%m.ColCnt()]
	}
	return f
}

func (m *Mat) ToString() string {
	mStr := strings.Builder{}
	for i := range *m {
		if i > 0 {
			mStr.WriteString("\n")
		}
		mStr.WriteString(fmt.Sprintf("%v", (*m)[i]))
	}
	return mStr.String()
}

func (m *Mat) Describe() string {
	return fmt.Sprintf(
		"%dx%d Matrix, %d Bytes in memory:\n%s",
		m.RowCnt(), m.ColCnt(), m.ByteSize(), m.ToString(),
	)
}
