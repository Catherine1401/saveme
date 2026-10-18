package update

import (
	"bytes"
	"encoding/binary"
	"os"
	"runtime"
)

func validBinary(data []byte) bool {
	const pePointer = 60
	const signatureSize = 4
	const peMachine = 4
	const elfX86 = 62
	const peX86 = 0x8664
	const mz, pe, mzSize = "MZ", "PE\x00\x00", 2
	const windows, amd64, arm64, header, zero = "windows", "amd64", "arm64", 64, 0
	if len(data) < header {
		return false
	}
	machine := uint16(elfX86)
	if runtime.GOARCH == arm64 {
		const elfArm = 183
		machine = elfArm
	} else if runtime.GOARCH != amd64 {
		return false
	}
	if runtime.GOOS != windows {
		const elfMachine = 18
		const elfClass = 2
		const littleEndian = 1
		const magic, classIndex, endianIndex = "\x7fELF", 4, 5
		return bytes.Equal(data[:signatureSize], []byte(magic)) && data[classIndex] == elfClass && data[endianIndex] == littleEndian && binary.LittleEndian.Uint16(data[elfMachine:]) == machine
	}
	machine = peX86
	if runtime.GOARCH == arm64 {
		const peArm = 0xaa64
		machine = peArm
	}

	offset := uint64(binary.LittleEndian.Uint32(data[pePointer:]))
	if offset+peMachine+mzSize > uint64(len(data)) {
		return false
	}
	return bytes.Equal(data[zero:mzSize], []byte(mz)) && bytes.Equal(data[offset:offset+signatureSize], []byte(pe)) && binary.LittleEndian.Uint16(data[offset+peMachine:]) == machine
}

func checkBinary(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !validBinary(data) {
		return os.ErrInvalid
	}
	return nil
}
