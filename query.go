package duckdb_go_bindings

/*
#include <duckdb.h>
#include <stdlib.h>
#include <string.h>
#include <duckdb_go_bindings.h>
*/
import "C"

import "unsafe"

// Query wraps duckdb_query.
// outRes must be destroyed with DestroyResult.
func Query(conn Connection, query string, outRes *Result) State {
	if debugMode {
		incrAllocCount("res")
	}
	cQuery := C.CString(query)
	defer Free(unsafe.Pointer(cQuery))

	return C.duckdb_query(conn.data(), cQuery, &outRes.data)
}

// DestroyResult wraps duckdb_destroy_result.
func DestroyResult(res *Result) {
	if res == nil || res.data.internal_data == nil {
		return
	}
	if debugMode {
		decrAllocCount("res")
	}
	C.duckdb_destroy_result(&res.data)
	res.data.internal_data = nil
}

func ColumnName(res *Result, col IdxT) string {
	name := C.duckdb_column_name(&res.data, col)
	return C.GoString(name)
}

func ColumnType(res *Result, col IdxT) Type {
	return C.duckdb_column_type(&res.data, col)
}

func ResultStatementType(res Result) StatementType {
	return C.duckdb_result_statement_type(res.data)
}

// ColumnLogicalType wraps duckdb_column_logical_type.
// The return value must be destroyed with DestroyLogicalType.
func ColumnLogicalType(res *Result, col IdxT) LogicalType {
	logicalType := C.duckdb_column_logical_type(&res.data, col)
	if debugMode {
		incrAllocCount("logicalType")
	}
	return LogicalType{
		Ptr: unsafe.Pointer(logicalType),
	}
}

// ResultGetArrowOptions wraps duckdb_result_get_arrow_options.
// The return value must be destroyed with DestroyArrowOptions.
func ResultGetArrowOptions(res *Result) ArrowOptions {
	options := C.duckdb_result_get_arrow_options(&res.data)
	if debugMode {
		incrAllocCount("arrowOptions")
	}
	return ArrowOptions{
		Ptr: unsafe.Pointer(options),
	}
}

func ColumnCount(res *Result) IdxT {
	return C.duckdb_column_count(&res.data)
}

func RowsChanged(res *Result) IdxT {
	return C.duckdb_rows_changed(&res.data)
}

func ResultError(res *Result) string {
	err := C.duckdb_result_error(&res.data)
	return C.GoString(err)
}

func ResultErrorType(res *Result) ErrorType {
	return C.duckdb_result_error_type(&res.data)
}

// ResultGetChunk wraps duckdb_result_get_chunk.
// The return value must be destroyed with DestroyDataChunk.
// Deprecated: See C API documentation.
func ResultGetChunk(res Result, index IdxT) DataChunk {
	chunk := C.duckdb_result_get_chunk(res.data, index)
	if debugMode {
		incrAllocCount("chunk")
	}
	return DataChunk{
		Ptr: unsafe.Pointer(chunk),
	}
}

// ResultIsStreaming wraps duckdb_result_is_streaming.
// Deprecated: ResultIsStreaming is deprecated.
func ResultIsStreaming(res Result) bool {
	return bool(C.duckdb_result_is_streaming(res.data))
}

// Deprecated: See C API documentation.
func ResultChunkCount(res Result) IdxT {
	return C.duckdb_result_chunk_count(res.data)
}

func ResultReturnType(res Result) ResultType {
	return C.duckdb_result_return_type(res.data)
}

// StreamFetchChunk wraps duckdb_stream_fetch_chunk.
// Returns a data chunk from the streaming result.
// The returned data chunk must be destroyed with DestroyDataChunk.
// Returns a data chunk with size 0 when the result is exhausted.
// Deprecated: StreamFetchChunk is deprecated.
func StreamFetchChunk(res Result, outChunk *DataChunk) State {
	chunk := C.duckdb_stream_fetch_chunk(res.data)
	// duckdb_stream_fetch_chunk returns NULL if the result has an error.
	if chunk == nil {
		return StateError
	}
	outChunk.Ptr = unsafe.Pointer(chunk)
	if debugMode {
		incrAllocCount("chunk")
	}
	return StateSuccess
}

func FetchChunk(res Result) DataChunk {
	chunk := C.duckdb_fetch_chunk(res.data)
	if debugMode && chunk != nil {
		incrAllocCount("chunk")
	}
	return DataChunk{
		Ptr: unsafe.Pointer(chunk),
	}
}

// Deprecated: See C API documentation.
func ValueInt64(res *Result, col IdxT, row IdxT) int64 {
	v := C.duckdb_value_int64(&res.data, col, row)
	return int64(v)
}
