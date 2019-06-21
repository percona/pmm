package parser

/*
#cgo CFLAGS: -Iinclude -g -fstack-protector
#cgo LDFLAGS:
#include "pg_query.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

func init() {
	C.pg_query_init()
}

// ParseToJSON - Parses the given SQL statement into an AST (JSON format)
func ParseToJSON(input string) (result string, err error) {
	inputC := C.CString(input)
	defer C.free(unsafe.Pointer(inputC))

	resultC := C.pg_query_parse(inputC)

	defer C.pg_query_free_parse_result(resultC)

	if resultC.error != nil {
		errMessage := C.GoString(resultC.error.message)
		err = errors.New(errMessage)
		return
	}

	result = C.GoString(resultC.parse_tree)

	return
}

// ParsePlPgSqlToJSON - Parses the given PL/pgSQL function statement into an AST (JSON format)
func ParsePlPgSqlToJSON(input string) (result string, err error) {
	inputC := C.CString(input)
	defer C.free(unsafe.Pointer(inputC))

	resultC := C.pg_query_parse_plpgsql(inputC)

	defer C.pg_query_free_plpgsql_parse_result(resultC)

	if resultC.error != nil {
		errMessage := C.GoString(resultC.error.message)
		err = errors.New(errMessage)
		return
	}

	result = C.GoString(resultC.plpgsql_funcs)

	return
}

// Normalize the passed SQL statement to replace constant values with ? characters
func Normalize(input string) (result string, err error) {
	inputC := C.CString(input)
	defer C.free(unsafe.Pointer(inputC))

	resultC := C.pg_query_normalize(inputC)
	defer C.pg_query_free_normalize_result(resultC)

	if resultC.error != nil {
		errMessage := C.GoString(resultC.error.message)
		err = errors.New(errMessage)
		return
	}

	result = C.GoString(resultC.normalized_query)

	return
}

// FastFingerprint - Fingerprint the passed SQL statement using the C extension
func FastFingerprint(input string) (result string, err error) {
	inputC := C.CString(input)
	defer C.free(unsafe.Pointer(inputC))

	resultC := C.pg_query_fingerprint(inputC)
	defer C.pg_query_free_fingerprint_result(resultC)

	if resultC.error != nil {
		errMessage := C.GoString(resultC.error.message)
		err = errors.New(errMessage)
		return
	}

	result = C.GoString(resultC.hexdigest)

	return
}
