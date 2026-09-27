//go:build !(js && wasm)

package main

import "os"

// lookupConfig returns the environment variable as is.
func lookupConfig(name string) (string, bool) {
	return os.LookupEnv(name)
}
