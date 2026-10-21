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
