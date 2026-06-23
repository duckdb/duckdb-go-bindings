package duckdb_go_bindings

import (
	"testing"
	"unsafe"
)

func withIsolatedAllocationCounts(t *testing.T) {
	t.Helper()

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
	if !ok || got != 1 {
		t.Fatalf("GetAllocationCount(%q) = %d, %t; want 1, true", AllocationCounterValue, got, ok)
	}

	got, ok = GetAllocationCount(AllocationCounterDatabase)
	if !ok || got != 1 {
		t.Fatalf("GetAllocationCount(%q) = %d, %t; want 1, true", AllocationCounterDatabase, got, ok)
	}

	got, ok = GetAllocationCount("value")
	if ok || got != 0 {
		t.Fatalf("GetAllocationCount(\"value\") = %d, %t; want 0, false", got, ok)
	}

	const want = "db count is 1\nv count is 1\n"
	if got := GetAllocationCounts(); got != want {
		t.Fatalf("GetAllocationCounts() = %q; want %q", got, want)
	}
}

func TestDecrAllocationCountAbsentIsNoOp(t *testing.T) {
	withIsolatedAllocationCounts(t)

	decrAllocationCount(valueAllocation)
	if got := GetAllocationCounts(); got != "" {
		t.Fatalf("GetAllocationCounts() after absent decrement = %q; want empty", got)
	}

	incrAllocationCount(valueAllocation)
	decrAllocationCount(valueAllocation)
	decrAllocationCount(valueAllocation)

	got, ok := GetAllocationCount(AllocationCounterValue)
	if ok || got != 0 {
		t.Fatalf("GetAllocationCount(%q) after double decrement = %d, %t; want 0, false", AllocationCounterValue, got, ok)
	}
}

func TestTrackAllocationHonorsDebugMode(t *testing.T) {
	withIsolatedAllocationCounts(t)

	var marker byte
	trackAllocation(valueAllocation, unsafe.Pointer(&marker))

	got, ok := GetAllocationCount(AllocationCounterValue)
	if debugMode {
		if !ok || got != 1 {
			t.Fatalf("GetAllocationCount(%q) = %d, %t; want 1, true", AllocationCounterValue, got, ok)
		}
		return
	}

	if ok || got != 0 {
		t.Fatalf("GetAllocationCount(%q) = %d, %t; want 0, false", AllocationCounterValue, got, ok)
	}
}

func TestTrackAllocationIgnoresNilPointer(t *testing.T) {
	withIsolatedAllocationCounts(t)

	trackAllocation(valueAllocation, nil)

	got, ok := GetAllocationCount(AllocationCounterValue)
	if ok || got != 0 {
		t.Fatalf("GetAllocationCount(%q) = %d, %t; want 0, false", AllocationCounterValue, got, ok)
	}
}

func TestTrackedResultIgnoresEmptyResult(t *testing.T) {
	withIsolatedAllocationCounts(t)

	trackedResult(&Result{})

	got, ok := GetAllocationCount(AllocationCounterResult)
	if ok || got != 0 {
		t.Fatalf("GetAllocationCount(%q) = %d, %t; want 0, false", AllocationCounterResult, got, ok)
	}
}
