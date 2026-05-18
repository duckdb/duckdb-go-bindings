package duckdb_go_bindings

// UTF8Bytes is a UTF-8 byte slice for zero-copy VARCHAR / JSON appender columns.
// Use with duckdb-go AppendRow via duckdb.AppendBytes, or VectorAssignByteElement.
type UTF8Bytes []byte

// UnsafeUTF8Bytes is UTF-8 for UnsafeVectorAssignStringElementLen (no UTF-8 check).
// The backing slice must remain valid until the Appender chunk is flushed.
type UnsafeUTF8Bytes []byte

// Bytes returns the underlying slice without allocation.
func (b UTF8Bytes) Bytes() []byte { return []byte(b) }

// Bytes returns the underlying slice without allocation.
func (b UnsafeUTF8Bytes) Bytes() []byte { return []byte(b) }

// VectorAssignUTF8Bytes writes blob using either the copying or zero-copy API.
// Prefer VectorAssignByteElement / VectorAssignStringElementLen directly when possible.
func VectorAssignUTF8Bytes(vec Vector, index IdxT, blob []byte, validateUTF8 bool) {
	if validateUTF8 {
		VectorAssignByteElement(vec, index, blob)
		return
	}
	UnsafeVectorAssignStringElementLen(vec, index, blob)
}
