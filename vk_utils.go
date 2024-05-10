package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Provides general helper functions for comparisons and conversions

func allOfAinB(a []string, b []string) bool {
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

func rawBytes(p interface{}) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, p)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

func terminatedStr(s string) string {
	if s[len(s)-1] != '\x00' {
		return s + "\x00"
	}
	return s
}

func terminatedStrs(strs []string) []string {
	for i := range strs {
		strs[i] = terminatedStr(strs[i])
	}
	return strs
}

// Nasty conversion logic taken from: https://github.com/vulkan-go/asche/blob/master/util.go
// it should be equivalent to C++ 'reinterpret_cast<const uint32_t*>(code.data());'
// See: https://vulkan-tutorial.com/Drawing_a_triangle/Graphics_pipeline_basics/Shader_modules
func sliceUint32(data []byte) []uint32 {
	const m = 0x7fffffff
	return (*[m / 4]uint32)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&data)).Data))[:len(data)/4]
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
