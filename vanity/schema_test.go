package vanity

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// The schema and Handler must agree: every config under testdata/valid passes
// both, every one under testdata/invalid fails both.
func TestSchema(t *testing.T) {
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	schema, err := c.Compile("../config.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []bool{true, false} {
		dir := map[bool]string{true: "valid", false: "invalid"}[want]
		files, _ := filepath.Glob(filepath.Join("testdata", dir, "*.json"))
		if len(files) == 0 {
			t.Fatalf("no testdata/%s", dir)
		}
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(doc); (err == nil) != want {
				t.Errorf("%s: schema: %v", f, err)
			}
			var cfg Config
			if err := json.Unmarshal(b, &cfg); err != nil {
				t.Fatalf("%s: %v", f, err)
			}
			if _, err := Handler(cfg); (err == nil) != want {
				t.Errorf("%s: Handler: %v", f, err)
			}
		}
	}
}
