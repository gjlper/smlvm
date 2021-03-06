package test

import (
	"shanhu.io/smlvm/pl"
)

// Fuzz implements go-fuzz interface.
func Fuzz(data []byte) int {
	_, errs, _ := pl.CompileSingle("test.g", string(data), false)
	if len(errs) > 0 {
		return 0
	}
	return 1
}
