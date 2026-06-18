package duckdb_go_bindings

/*
#include <duckdb.h>
#include <stdlib.h>
#include <string.h>
#include <duckdb_go_bindings.h>
*/
import "C"

import "unsafe"

// CreateInstanceCache wraps duckdb_create_instance_cache.
// The return value must be destroyed with DestroyInstanceCache.
func CreateInstanceCache() InstanceCache {
	cache := C.duckdb_create_instance_cache()
	if debugMode {
		incrAllocCount("cache")
	}
	return InstanceCache{
		Ptr: unsafe.Pointer(cache),
	}
}

// GetOrCreateFromCache wraps duckdb_get_or_create_from_cache.
// outDb must be closed with Close.
func GetOrCreateFromCache(cache InstanceCache, path string, outDb *Database, config Config, errMsg *string) State {
	cPath := C.CString(path)
	defer Free(unsafe.Pointer(cPath))
	var err *C.char
	defer func() { Free(unsafe.Pointer(err)) }()

	var db C.duckdb_database
	state := C.duckdb_get_or_create_from_cache(cache.data(), cPath, &db, config.data(), &err)
	outDb.Ptr = unsafe.Pointer(db)
	*errMsg = C.GoString(err)

	if debugMode {
		incrAllocCount("db")
	}
	return state
}

// DestroyInstanceCache wraps duckdb_destroy_instance_cache.
func DestroyInstanceCache(cache *InstanceCache) {
	if cache.Ptr == nil {
		return
	}
	if debugMode {
		decrAllocCount("cache")
	}
	data := cache.data()
	C.duckdb_destroy_instance_cache(&data)
	cache.Ptr = nil
}

// Open wraps duckdb_open.
// outDb must be closed with Close.
// Deprecated: Use OpenExt.
func Open(path string, outDb *Database) State {
	cPath := C.CString(path)
	defer Free(unsafe.Pointer(cPath))

	var db C.duckdb_database
	state := C.duckdb_open(cPath, &db)
	outDb.Ptr = unsafe.Pointer(db)

	if debugMode {
		incrAllocCount("db")
	}
	return state
}

// OpenExt wraps duckdb_open_ext.
// outDb must be closed with Close.
func OpenExt(path string, outDb *Database, config Config, errMsg *string) State {
	cPath := C.CString(path)
	defer Free(unsafe.Pointer(cPath))
	var err *C.char
	defer func() { Free(unsafe.Pointer(err)) }()

	var db C.duckdb_database
	state := C.duckdb_open_ext(cPath, &db, config.data(), &err)
	outDb.Ptr = unsafe.Pointer(db)
	*errMsg = C.GoString(err)

	if debugMode {
		incrAllocCount("db")
	}
	return state
}

// Close wraps duckdb_close.
func Close(db *Database) {
	if db.Ptr == nil {
		return
	}
	if debugMode {
		decrAllocCount("db")
	}
	data := db.data()
	C.duckdb_close(&data)
	db.Ptr = nil
}

// Connect wraps duckdb_connect.
// outConn must be disconnected with Disconnect.
func Connect(db Database, outConn *Connection) State {
	var conn C.duckdb_connection
	state := C.duckdb_connect(db.data(), &conn)
	outConn.Ptr = unsafe.Pointer(conn)
	if debugMode {
		incrAllocCount("conn")
	}
	return state
}

func Interrupt(conn Connection) {
	C.duckdb_interrupt(conn.data())
}

func QueryProgress(conn Connection) QueryProgressType {
	return C.duckdb_query_progress(conn.data())
}

// Disconnect wraps duckdb_disconnect.
func Disconnect(conn *Connection) {
	if conn.Ptr == nil {
		return
	}
	if debugMode {
		decrAllocCount("conn")
	}
	data := conn.data()
	C.duckdb_disconnect(&data)
	conn.Ptr = nil
}

// ConnectionGetClientContext wraps duckdb_connection_get_client_context.
// outCtx must be destroyed with DestroyClientContext.
func ConnectionGetClientContext(conn Connection, outCtx *ClientContext) {
	var ctx C.duckdb_client_context
	C.duckdb_connection_get_client_context(conn.data(), &ctx)
	outCtx.Ptr = unsafe.Pointer(ctx)
	if debugMode {
		incrAllocCount("ctx")
	}
}

// ConnectionGetArrowOptions wraps duckdb_connection_get_arrow_options.
// outOptions must be destroyed with DestroyArrowOptions.
func ConnectionGetArrowOptions(conn Connection, outOptions *ArrowOptions) {
	var options C.duckdb_arrow_options
	C.duckdb_connection_get_arrow_options(conn.data(), &options)
	outOptions.Ptr = unsafe.Pointer(options)
	if debugMode {
		incrAllocCount("arrowOptions")
	}
}

func ClientContextGetConnectionId(ctx ClientContext) IdxT {
	return C.duckdb_client_context_get_connection_id(ctx.data())
}

// DestroyClientContext wraps duckdb_destroy_client_context.
func DestroyClientContext(ctx *ClientContext) {
	if ctx.Ptr == nil {
		return
	}
	if debugMode {
		decrAllocCount("ctx")
	}
	data := ctx.data()
	C.duckdb_destroy_client_context(&data)
	ctx.Ptr = nil
}

// DestroyArrowOptions wraps duckdb_destroy_arrow_options.
func DestroyArrowOptions(options *ArrowOptions) {
	if options.Ptr == nil {
		return
	}
	if debugMode {
		decrAllocCount("arrowOptions")
	}
	data := options.data()
	C.duckdb_destroy_arrow_options(&data)
	options.Ptr = nil
}

func LibraryVersion() string {
	cStr := C.duckdb_library_version()
	return C.GoString(cStr)
}

// GetTableNames wraps duckdb_get_table_names.
// The return value must be destroyed with DestroyValue.
func GetTableNames(conn Connection, query string, qualified bool) Value {
	cQuery := C.CString(query)
	defer Free(unsafe.Pointer(cQuery))
	v := C.duckdb_get_table_names(conn.data(), cQuery, C.bool(qualified))
	if debugMode {
		incrAllocCount("v")
	}
	return Value{
		Ptr: unsafe.Pointer(v),
	}
}
