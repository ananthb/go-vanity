//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/syumai/workers-go/cloudflare"
)

// getenvJSON reads a binding as JSON. A json binding, or a table under [vars]
// in wrangler.toml, reaches the Worker as an object; a plain var as a string.
func getenvJSON(name string) string {
	v := cloudflare.GetBinding(name)
	switch v.Type() {
	case js.TypeUndefined:
		return "null"
	case js.TypeString:
		return v.String()
	default:
		return js.Global().Get("JSON").Call("stringify", v).String()
	}
}
