package duckdb_go_bindings

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func withIsolatedAllocationCounts(t *testing.T) {
	t.Helper()

	// Tests using this helper must stay serial: it swaps package-global
	// allocation state while the test is running.
	allocCounts.lock.Lock()
	previous := allocCounts.m
	allocCounts.m = nil
	allocCounts.lock.Unlock()

	t.Cleanup(func() {
		allocCounts.lock.Lock()
		allocCounts.m = previous
		allocCounts.lock.Unlock()
	})
}

func TestAllocationCountersUseStablePublicKeys(t *testing.T) {
	withIsolatedAllocationCounts(t)

	incrAllocationCount(valueAllocation)
	incrAllocationCount(databaseAllocation)

	got, ok := GetAllocationCount(AllocationCounterValue)
	require.True(t, ok)
	require.Equal(t, 1, got)

	got, ok = GetAllocationCount(AllocationCounterDatabase)
	require.True(t, ok)
	require.Equal(t, 1, got)

	got, ok = GetAllocationCount("value")
	require.False(t, ok)
	require.Zero(t, got)

	const want = "db count is 1\nv count is 1\n"
	require.Equal(t, want, GetAllocationCounts())
}

func TestDecrAllocationCountAbsentIsNoOp(t *testing.T) {
	withIsolatedAllocationCounts(t)

	decrAllocationCount(valueAllocation)
	require.Empty(t, GetAllocationCounts())

	incrAllocationCount(valueAllocation)
	decrAllocationCount(valueAllocation)
	decrAllocationCount(valueAllocation)

	got, ok := GetAllocationCount(AllocationCounterValue)
	require.False(t, ok)
	require.Zero(t, got)
}

func TestTrackAllocationHonorsDebugMode(t *testing.T) {
	withIsolatedAllocationCounts(t)

	var marker byte
	trackAllocation(valueAllocation, unsafe.Pointer(&marker))

	got, ok := GetAllocationCount(AllocationCounterValue)
	if debugMode {
		require.True(t, ok)
		require.Equal(t, 1, got)
		return
	}

	require.False(t, ok)
	require.Zero(t, got)
}

func TestTrackAllocationIgnoresNilPointer(t *testing.T) {
	withIsolatedAllocationCounts(t)

	trackAllocation(valueAllocation, nil)

	got, ok := GetAllocationCount(AllocationCounterValue)
	require.False(t, ok)
	require.Zero(t, got)
}

func TestTrackedResultIgnoresEmptyResult(t *testing.T) {
	withIsolatedAllocationCounts(t)

	trackedResult(&Result{})

	got, ok := GetAllocationCount(AllocationCounterResult)
	require.False(t, ok)
	require.Zero(t, got)
}

func TestPackedNames(t *testing.T) {
	defer VerifyAllocationCounters()
	typ := CreateEnumType([]string{"", "plain", "Grüße", "nul\x00suffix"})
	defer DestroyLogicalType(&typ)
	for i, want := range []string{"", "plain", "Grüße", "nul"} {
		require.Equal(t, want, EnumDictionaryValue(typ, IdxT(i)))
	}
}

func TestPackedLogicalTypes(t *testing.T) {
	defer VerifyAllocationCounters()
	integerT := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&integerT)
	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)
	typ := CreateStructType([]LogicalType{integerT, varcharT}, []string{"number", "text"})
	defer DestroyLogicalType(&typ)
	for i, want := range []Type{TypeInteger, TypeVarchar} {
		child := StructTypeChildType(typ, IdxT(i))
		require.Equal(t, want, GetTypeId(child))
		DestroyLogicalType(&child)
	}
	require.Equal(t, "number", StructTypeChildName(typ, 0))
	require.Equal(t, "text", StructTypeChildName(typ, 1))
}
