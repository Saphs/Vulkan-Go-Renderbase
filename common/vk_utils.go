package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"local/vector_math"
	"unsafe"
)

// Provides general helper functions for comparisons and conversions

// AllOfAinB comparison function to ensure a given list is fully contains in another. This is
// mainly used to check for extension and layer support during the initialization process.
func AllOfAinB(a []string, b []string) bool {
	for _, _a := range a {
		isIn := false
		for _, _b := range b {
			if _a == _b {
				isIn = true
				break
			}
		}
		if !isIn {
			return false
		}
	}
	return true
}

// RawBytes writes a given object as its byte representation voiding all type information in the process
// this is mainly used to be able to put data into vk.Memcopy
func RawBytes(p interface{}) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, p)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

// ToByteArr drops type reference from float array to allow Go to pass an unsafe.Pointer to Vulkan
func ToByteArr(in []float32) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&in[0])), len(in)*4)
}

func UnsafeMatPtr(m *vector_math.Mat) unsafe.Pointer {
	return unsafe.Pointer(&ToByteArr(m.Unroll())[0])
}

// TerminatedStr ensures the given string is \x00 terminated as vulkan expects this in certain structs
func TerminatedStr(s string) string {
	if s[len(s)-1] != '\x00' {
		return s + "\x00"
	}
	return s
}

func TerminatedStrs(strs []string) []string {
	for i := range strs {
		strs[i] = TerminatedStr(strs[i])
	}
	return strs
}

// AsUint32Arr Casts a []byte to []uint32 using nasty conversion logic taken from:
// https://github.com/vulkan-go/asche/blob/master/util.go and is only used to construct shader modules.
// It should be equivalent to C++ 'reinterpret_cast<const uint32_t*>(code.data());'
// See: https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Shader_modules
func AsUint32Arr(data []byte) []uint32 {
	const m = 0x7fffffff
	return (*[m / 4]uint32)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&data)).Data))[:len(data)/4]
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
