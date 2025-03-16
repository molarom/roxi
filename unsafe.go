// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"unsafe"
)

// toBytes converts a string to bytes avoiding allocation.
func toBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}

	// copied from os.File.WriteString
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// toString converts a []byte to a string avoiding allocation.
func toString(bytes []byte) string {
	// copied from strings.Builder.String
	return unsafe.String(unsafe.SliceData(bytes), len(bytes))
}
