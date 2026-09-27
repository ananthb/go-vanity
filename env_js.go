//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/syumai/workers-go/cloudflare"
)

// lookupConfig returns a binding as text. A json binding, or a table under
// [vars] in wrangler.toml, reaches the Worker as an object and is serialized;
// a plain var is returned as is.
func lookupConfig(name string) (string, bool) {
	v := cloudflare.GetBinding(name)
	switch v.Type() {
	case js.TypeUndefined, js.TypeNull:
		return "", false
	case js.TypeString:
		return v.String(), true
	default:
		return js.Global().Get("JSON").Call("stringify", v).String(), true
	}
}
