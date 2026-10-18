package update

import (
	"encoding/binary"
	"runtime"
	"testing"
)

func testBinary() []byte {
	const elfMachine = 18
	const elfX86 = 62
	const class = 2
	const endian = 1
	const classIndex = 4
	const endianIndex = 5
	const magic = "\x7fELF"
	const size, windows, arm64 = 128, "windows", "arm64"
	data := make([]byte, size)

	copy(data, magic)
	data[classIndex], data[endianIndex] = class, endian
	machine := uint16(elfX86)
	if runtime.GOARCH == arm64 {
		const elfArm = 183
		machine = elfArm
	}
	binary.LittleEndian.PutUint16(data[elfMachine:], machine)
	if runtime.GOOS == windows {
		const mz = "MZ"
		const pe = "PE\x00\x00"
		const pointer = 60
		const offset = 64
		const machineOffset = 4
		const x86 = 0x8664
		copy(data, mz)
		copy(data[offset:], pe)
		binary.LittleEndian.PutUint32(data[pointer:], offset)
		machine = x86
		if runtime.GOARCH == arm64 {
			const arm = 0xaa64
			machine = arm
		}
		binary.LittleEndian.PutUint16(data[offset+machineOffset:], machine)
	}
	return data
}

func TestExecutableArchitectureAndCorruption(t *testing.T) {
	const bad = "bad executable"
	const machine = 18
	const windows = "windows"
	if !validBinary(testBinary()) || validBinary([]byte(bad)) {
		t.Fatal(bad)
	}
	data := testBinary()
	index := machine
	if runtime.GOOS == windows {
		const literal1 = 4
		const pointer = 60
		index = int(binary.LittleEndian.Uint32(data[pointer:])) + literal1
	}
	data[index]++
	if validBinary(data) {
		t.Fatal(bad)
	}
}
