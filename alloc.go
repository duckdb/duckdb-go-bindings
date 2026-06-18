package duckdb_go_bindings

/*
#include <duckdb.h>
#include <stdlib.h>
#include <string.h>
#include <duckdb_go_bindings.h>
*/
import "C"

import (
	"fmt"
	"log"
	"sync"
	"unsafe"
)

// TODO:
// duckdb_malloc

func Free(ptr unsafe.Pointer) {
	C.duckdb_free(ptr)
}

func ValidityMaskValueIsValid(maskPtr unsafe.Pointer, index IdxT) bool {
	castMaskPtr := (*C.uint64_t)(maskPtr)
	return bool(C.duckdb_go_bindings_is_valid(castMaskPtr, index))
}

const (
	logicalTypeSize = C.size_t(unsafe.Sizeof((C.duckdb_logical_type)(nil)))
	valueSize       = C.size_t(unsafe.Sizeof((C.duckdb_value)(nil)))
	charSize        = C.size_t(unsafe.Sizeof((*C.char)(nil)))
)

// The return value must be freed with Free.
func allocLogicalTypes(types []LogicalType) *C.duckdb_logical_type {
	count := len(types)
	typesPtr := (*C.duckdb_logical_type)(C.calloc(C.size_t(count), logicalTypeSize))

	for i, t := range types {
		C.duckdb_go_bindings_set_logical_type(typesPtr, t.data(), IdxT(i))
	}

	return typesPtr
}

// The return value must be freed with Free.
func allocValues(values []Value) *C.duckdb_value {
	count := len(values)
	valuesPtr := (*C.duckdb_value)(C.calloc(C.size_t(count), valueSize))

	for i, val := range values {
		C.duckdb_go_bindings_set_value(valuesPtr, val.data(), IdxT(i))
	}

	return valuesPtr
}

// nameListAlloc is produced by allocNames: arr[i] point into NUL-padded blob backing.
type nameListAlloc struct {
	arr  **C.char
	blob *C.char
}

func freeNameList(a nameListAlloc) {
	if a.blob != nil {
		Free(unsafe.Pointer(a.blob))
	}
	if a.arr != nil {
		Free(unsafe.Pointer(a.arr))
	}
}

// The return values must be released with freeNameList.
func allocNames(names []string) nameListAlloc {
	n := len(names)
	if n == 0 {
		return nameListAlloc{}
	}

	var blobSize C.size_t
	for _, s := range names {
		blobSize += C.size_t(len(s)) + 1
	}

	arrMem := unsafe.Pointer(C.duckdb_malloc(C.size_t(n) * charSize))
	if arrMem == nil {
		panic("duckdb-go-bindings: duckdb_malloc name array returned nil")
	}
	arr := (**C.char)(arrMem)

	blobMem := unsafe.Pointer(C.duckdb_malloc(blobSize))
	if blobMem == nil {
		Free(arrMem)
		panic("duckdb-go-bindings: duckdb_malloc name blob returned nil")
	}
	blob := (*C.char)(blobMem)

	var off uintptr
	for i := 0; i < n; i++ {
		s := names[i]
		slen := len(s)
		dest := unsafe.Add(unsafe.Pointer(blob), off)
		if slen > 0 {
			C.memcpy(unsafe.Pointer(dest), unsafe.Pointer(unsafe.StringData(s)), C.size_t(slen))
		}
		*(*byte)(unsafe.Add(dest, uintptr(slen))) = 0

		slot := (**C.char)(unsafe.Add(unsafe.Pointer(arr), uintptr(i)*uintptr(charSize)))
		*slot = (*C.char)(dest)

		off += uintptr(slen + 1)
	}

	return nameListAlloc{arr: arr, blob: blob}
}

var allocCounts syncMap

type syncMap struct {
	lock sync.Mutex
	m    map[string]int
}

func incrAllocCount(k string) {
	allocCounts.lock.Lock()
	defer allocCounts.lock.Unlock()

	if allocCounts.m == nil {
		allocCounts.m = make(map[string]int)
	}

	allocCounts.m[k]++
}

func decrAllocCount(k string) {
	allocCounts.lock.Lock()
	defer allocCounts.lock.Unlock()

	if allocCounts.m == nil {
		allocCounts.m = make(map[string]int)
	}

	if v, ok := allocCounts.m[k]; ok {
		if v == 1 {
			delete(allocCounts.m, k)
			return
		}
		allocCounts.m[k]--
	}
}

// VerifyAllocationCounters verifies all allocation counters.
// This includes the instance cache, which should be kept alive as long as the application is kept alive,
// causing this verification to fail.
// If you're using the instance cache, use VerifyAllocationCounter instead.
func VerifyAllocationCounters() {
	msg := GetAllocationCounts()
	if msg != "" {
		log.Panic(msg)
	}
}

// GetAllocationCount returns the value of an allocation count, and true,
// if it exists, otherwise zero, and false.
func GetAllocationCount(k string) (int, bool) {
	allocCounts.lock.Lock()
	defer allocCounts.lock.Unlock()

	if allocCounts.m == nil {
		return 0, false
	}

	v, ok := allocCounts.m[k]
	return v, ok
}

// GetAllocationCounts returns the value of each non-zero allocation count.
func GetAllocationCounts() string {
	allocCounts.lock.Lock()
	defer allocCounts.lock.Unlock()

	if allocCounts.m == nil {
		return ""
	}

	msg := ""
	if len(allocCounts.m) != 0 {
		for k, v := range allocCounts.m {
			msg += fmt.Sprintf("%s count is %d\n", k, v)
		}
	}

	return msg
}
