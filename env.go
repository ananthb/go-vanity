//go:build !(js && wasm)

package main

import "os"

func getenvJSON(name string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return "null"
}
