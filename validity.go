package duckdb_go_bindings

import "unsafe"

// validityBit locates row in DuckDB's validity mask, an array of uint64 words
// holding one bit per row.
func validityBit(maskPtr unsafe.Pointer, row IdxT) (word *uint64, bit uint64) {
	return (*uint64)(unsafe.Add(maskPtr, uintptr(row/64)*8)), 1 << (row % 64)
}

func ValidityMaskValueIsValid(maskPtr unsafe.Pointer, index IdxT) bool {
	word, bit := validityBit(maskPtr, index)
	return *word&bit != 0
}

func ValidityRowIsValid(maskPtr unsafe.Pointer, row IdxT) bool {
	return maskPtr == nil || ValidityMaskValueIsValid(maskPtr, row)
}

func ValiditySetRowValidity(maskPtr unsafe.Pointer, row IdxT, valid bool) {
	if valid {
		ValiditySetRowValid(maskPtr, row)
	} else {
		ValiditySetRowInvalid(maskPtr, row)
	}
}

func ValiditySetRowInvalid(maskPtr unsafe.Pointer, row IdxT) {
	if maskPtr != nil {
		word, bit := validityBit(maskPtr, row)
		*word &^= bit
	}
}

func ValiditySetRowValid(maskPtr unsafe.Pointer, row IdxT) {
	if maskPtr != nil {
		word, bit := validityBit(maskPtr, row)
		*word |= bit
	}
}
