package duckdb_go_bindings

/*
#include <duckdb.h>
#include <stdlib.h>
#include <string.h>
#include <duckdb_go_bindings.h>
*/
import "C"

import "unsafe"

// NOTE: No wrappings for function pointers.
// *duckdb_delete_callback_t
// *duckdb_copy_callback_t
// *duckdb_task_state
// *duckdb_scalar_function_bind_t
// *duckdb_scalar_function_t
// *duckdb_aggregate_state_size
// *duckdb_aggregate_init_t
// *duckdb_aggregate_destroy_t
// *duckdb_aggregate_update_t
// *duckdb_aggregate_combine_t
// *duckdb_aggregate_finalize_t
// *duckdb_table_function_bind_t
// *duckdb_table_function_init_t
// *duckdb_table_function_t
// *duckdb_cast_function_t
// *duckdb_replacement_callback_t
// *duckdb_logger_write_log_entry_t

// NOTE: We export the Ptr of each wrapped type pointer to allow (void *) typedef's of callback functions.
// See https://golang.org/issue/19837 and https://golang.org/issue/19835.

// NOTE: For some types (e.g., Appender, but not Config) omitting the Ptr causes
// the same somewhat mysterious runtime error as described for Result.
// 'runtime error: cgo argument has Go pointer to unpinned Go pointer'.
// See https://github.com/golang/go/issues/28606#issuecomment-2184269962.
// When using a type alias, duckdb_result itself contains a Go unsafe.Pointer for its 'void *internal_ptr' field.

// TODO:
// *duckdb_task_state

// Vector wraps *duckdb_vector.
type Vector struct {
	Ptr unsafe.Pointer
}

func (vec *Vector) data() C.duckdb_vector {
	return C.duckdb_vector(vec.Ptr)
}

// SelectionVector wraps *duckdb_selection_vector.
type SelectionVector struct {
	Ptr unsafe.Pointer
}

func (sel *SelectionVector) data() C.duckdb_selection_vector {
	return C.duckdb_selection_vector(sel.Ptr)
}

// InstanceCache wraps *duckdb_instance_cache.
type InstanceCache struct {
	Ptr unsafe.Pointer
}

func (cache *InstanceCache) data() C.duckdb_instance_cache {
	return C.duckdb_instance_cache(cache.Ptr)
}

// Database wraps *duckdb_database.
type Database struct {
	Ptr unsafe.Pointer
}

func (db *Database) data() C.duckdb_database {
	return C.duckdb_database(db.Ptr)
}

// Connection wraps *duckdb_connection.
type Connection struct {
	Ptr unsafe.Pointer
}

func (conn *Connection) data() C.duckdb_connection {
	return C.duckdb_connection(conn.Ptr)
}

// ClientContext wraps *duckdb_client_context.
type ClientContext struct {
	Ptr unsafe.Pointer
}

func (ctx *ClientContext) data() C.duckdb_client_context {
	return C.duckdb_client_context(ctx.Ptr)
}

// PreparedStatement wraps *duckdb_prepared_statement.
type PreparedStatement struct {
	Ptr unsafe.Pointer
}

func (preparedStmt *PreparedStatement) data() C.duckdb_prepared_statement {
	return C.duckdb_prepared_statement(preparedStmt.Ptr)
}

// ExtractedStatements wraps *duckdb_extracted_statements.
type ExtractedStatements struct {
	Ptr unsafe.Pointer
}

func (extractedStmts *ExtractedStatements) data() C.duckdb_extracted_statements {
	return C.duckdb_extracted_statements(extractedStmts.Ptr)
}

// PendingResult wraps *duckdb_pending_result.
type PendingResult struct {
	Ptr unsafe.Pointer
}

func (pendingRes *PendingResult) data() C.duckdb_pending_result {
	return C.duckdb_pending_result(pendingRes.Ptr)
}

// Appender wraps *duckdb_appender.
type Appender struct {
	Ptr unsafe.Pointer
}

func (appender *Appender) data() C.duckdb_appender {
	return C.duckdb_appender(appender.Ptr)
}

// TableDescription wraps *duckdb_table_description.
type TableDescription struct {
	Ptr unsafe.Pointer
}

func (description *TableDescription) data() C.duckdb_table_description {
	return C.duckdb_table_description(description.Ptr)
}

// Config wraps *duckdb_config.
type Config struct {
	Ptr unsafe.Pointer
}

func (config *Config) data() C.duckdb_config {
	return C.duckdb_config(config.Ptr)
}

// LogicalType wraps *duckdb_logical_type.
type LogicalType struct {
	Ptr unsafe.Pointer
}

func (logicalType *LogicalType) data() C.duckdb_logical_type {
	return C.duckdb_logical_type(logicalType.Ptr)
}

// CreateTypeInfo wraps *duckdb_create_type_info.
type CreateTypeInfo struct {
	Ptr unsafe.Pointer
}

func (info *CreateTypeInfo) data() C.duckdb_create_type_info {
	return C.duckdb_create_type_info(info.Ptr)
}

// DataChunk wraps *duckdb_data_chunk.
type DataChunk struct {
	Ptr unsafe.Pointer
}

func (chunk *DataChunk) data() C.duckdb_data_chunk {
	return C.duckdb_data_chunk(chunk.Ptr)
}

// Value wraps *duckdb_value.
type Value struct {
	Ptr unsafe.Pointer
}

func (v *Value) data() C.duckdb_value {
	return C.duckdb_value(v.Ptr)
}

// ProfilingInfo wraps *duckdb_profiling_info.
type ProfilingInfo struct {
	Ptr unsafe.Pointer
}

func (info *ProfilingInfo) data() C.duckdb_profiling_info {
	return C.duckdb_profiling_info(info.Ptr)
}

// ErrorData wraps *duckdb_error_data.
type ErrorData struct {
	Ptr unsafe.Pointer
}

func (errorData *ErrorData) data() C.duckdb_error_data {
	return C.duckdb_error_data(errorData.Ptr)
}

// Expression wraps *duckdb_expression.
type Expression struct {
	Ptr unsafe.Pointer
}

func (expr *Expression) data() C.duckdb_expression {
	return C.duckdb_expression(expr.Ptr)
}

// TODO:
// *duckdb_extension_info

// FunctionInfo wraps *duckdb_function_info.
type FunctionInfo struct {
	Ptr unsafe.Pointer
}

func (info *FunctionInfo) data() C.duckdb_function_info {
	return C.duckdb_function_info(info.Ptr)
}

// ScalarFunction wraps *duckdb_scalar_function.
type ScalarFunction struct {
	Ptr unsafe.Pointer
}

func (f *ScalarFunction) data() C.duckdb_scalar_function {
	return C.duckdb_scalar_function(f.Ptr)
}

// ScalarFunctionSet wraps *duckdb_scalar_function_set.
type ScalarFunctionSet struct {
	Ptr unsafe.Pointer
}

func (set *ScalarFunctionSet) data() C.duckdb_scalar_function_set {
	return C.duckdb_scalar_function_set(set.Ptr)
}

// TODO:
// *duckdb_aggregate_function
// *duckdb_aggregate_function_set
// *duckdb_aggregate_state

// TableFunction wraps *duckdb_table_function.
type TableFunction struct {
	Ptr unsafe.Pointer
}

func (f *TableFunction) data() C.duckdb_table_function {
	return C.duckdb_table_function(f.Ptr)
}

// BindInfo wraps *duckdb_bind_info.
type BindInfo struct {
	Ptr unsafe.Pointer
}

func (info *BindInfo) data() C.duckdb_bind_info {
	return C.duckdb_bind_info(info.Ptr)
}

// InitInfo wraps *C.duckdb_init_info.
type InitInfo struct {
	Ptr unsafe.Pointer
}

func (info *InitInfo) data() C.duckdb_init_info {
	return C.duckdb_init_info(info.Ptr)
}

// TODO:
// *duckdb_cast_function

// ReplacementScanInfo wraps *duckdb_replacement_scan.
type ReplacementScanInfo struct {
	Ptr unsafe.Pointer
}

func (info *ReplacementScanInfo) data() C.duckdb_replacement_scan_info {
	return C.duckdb_replacement_scan_info(info.Ptr)
}

// Arrow wraps *duckdb_arrow.
type Arrow struct {
	Ptr unsafe.Pointer
}

func (arrow *Arrow) data() C.duckdb_arrow {
	return C.duckdb_arrow(arrow.Ptr)
}

// ArrowStream wraps *duckdb_arrow_stream.
type ArrowStream struct {
	Ptr unsafe.Pointer
}

func (stream *ArrowStream) data() C.duckdb_arrow_stream {
	return C.duckdb_arrow_stream(stream.Ptr)
}

// ArrowSchema wraps *duckdb_arrow_schema.
type ArrowSchema struct {
	Ptr unsafe.Pointer
}

func (schema *ArrowSchema) data() C.duckdb_arrow_schema {
	return C.duckdb_arrow_schema(schema.Ptr)
}

// ArrowConvertedSchema wraps *duckdb_arrow_converted_schema.
type ArrowConvertedSchema struct {
	Ptr unsafe.Pointer
}

func (schema *ArrowConvertedSchema) data() C.duckdb_arrow_converted_schema {
	return C.duckdb_arrow_converted_schema(schema.Ptr)
}

// ArrowArray wraps *duckdb_arrow_array.
type ArrowArray struct {
	Ptr unsafe.Pointer
}

func (array *ArrowArray) data() C.duckdb_arrow_array {
	return C.duckdb_arrow_array(array.Ptr)
}

// ArrowOptions wraps *duckdb_arrow_options.
type ArrowOptions struct {
	Ptr unsafe.Pointer
}

func (options *ArrowOptions) data() C.duckdb_arrow_options {
	return C.duckdb_arrow_options(options.Ptr)
}

// LogStorage wraps *duckdb_log_storage.
type LogStorage struct {
	Ptr unsafe.Pointer
}

func (logStorage *LogStorage) data() C.duckdb_log_storage {
	return C.duckdb_log_storage(logStorage.Ptr)
}
