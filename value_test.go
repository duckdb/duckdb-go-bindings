package duckdb_go_bindings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimestampTZNS(t *testing.T) {
	defer VerifyAllocationCounters()

	for _, nanos := range []int64{0, -1234567890123456789, 1234567890123456789} {
		value := CreateTimestampTZNS(NewTimestampNS(nanos))
		defer DestroyValue(&value)
		require.Equal(t, TypeTimestampTZNS, GetTypeId(GetValueType(value)))
		got := GetTimestampTZNS(value)
		require.Equal(t, nanos, TimestampNSMembers(&got))
	}
}

func TestCreateStructValue(t *testing.T) {
	defer VerifyAllocationCounters()

	integerT := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&integerT)
	structT := CreateStructType([]LogicalType{integerT, integerT}, []string{"a", "b"})
	defer DestroyLogicalType(&structT)
	variantT := CreateLogicalType(TypeVariant)
	defer DestroyLogicalType(&variantT)

	first := CreateInt32(17)
	defer DestroyValue(&first)
	second := CreateInt32(42)
	defer DestroyValue(&second)

	for _, typ := range []LogicalType{structT, {}, integerT, variantT} {
		require.Nil(t, CreateStructValue(typ, []Value{first}).Ptr)
	}

	// Extra values are ignored.
	for _, values := range [][]Value{{first, second}, {first, second, {}}} {
		value := CreateStructValue(structT, values)
		defer DestroyValue(&value)
		require.NotNil(t, value.Ptr)
		for i, want := range []int32{17, 42} {
			child := GetStructChild(value, IdxT(i))
			defer DestroyValue(&child)
			require.Equal(t, want, GetInt32(child))
		}
	}
}

func TestCreateMapValue(t *testing.T) {
	defer VerifyAllocationCounters()

	integerT := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&integerT)
	mapT := CreateMapType(integerT, integerT)
	defer DestroyLogicalType(&mapT)

	entries := [][2]int32{{11, 101}, {29, 303}}
	var keys, values []Value
	for _, e := range entries {
		key := CreateInt32(e[0])
		defer DestroyValue(&key)
		value := CreateInt32(e[1])
		defer DestroyValue(&value)
		keys = append(keys, key)
		values = append(values, value)
	}

	require.Nil(t, CreateMapValue(mapT, keys, values[:1]).Ptr)
	require.Nil(t, CreateMapValue(mapT, keys[:1], values).Ptr)

	m := CreateMapValue(mapT, keys, values)
	defer DestroyValue(&m)
	require.NotNil(t, m.Ptr)
	require.Equal(t, IdxT(len(entries)), GetMapSize(m))
	for i, e := range entries {
		key := GetMapKey(m, IdxT(i))
		defer DestroyValue(&key)
		require.Equal(t, e[0], GetInt32(key))
		value := GetMapValue(m, IdxT(i))
		defer DestroyValue(&value)
		require.Equal(t, e[1], GetInt32(value))
	}
}
