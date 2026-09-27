// Command go-vanity serves vanity import paths for Go modules, as a Cloudflare
// Worker or as a plain HTTP server on $PORT (defaultc9900).
// Configuration is one JSON document passed as a CONFIG binding or through the environment.
package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/syumai/workers-go"
	"go.calculon.tech/go-vanity/vanity"
)

func config() (cfg vanity.Config, err error) {
	err = json.Unmarshal([]byte(getenvJSON("CONFIG")), &cfg)
	return
}

func main() {
	// Worker vars are only readable inside a request, so the handler is built
	// on the first one.
	handler := sync.OnceValues(func() (http.Handler, error) {
		cfg, err := config()
		if err != nil {
			return nil, err
		}
		return vanity.Handler(cfg)
	})
	workers.Serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, err := handler()
		if err != nil {
			http.Error(w, "config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		h.ServeHTTP(w, r)
	}))
}
