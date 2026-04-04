package schemas

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed *.schema.json
var FS embed.FS

// NewCompiler returns a jsonschema.Compiler with all embedded schemas pre-registered.
// Schemas are registered by their filename (e.g. "agents.schema.json"), which matches
// the $id field in each schema, enabling cross-schema $ref resolution.
func NewCompiler() (*jsonschema.Compiler, error) {
	c := jsonschema.NewCompiler()
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".schema.json") {
			return nil
		}
		data, readErr := FS.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("reading embedded schema %s: %w", path, readErr)
		}
		var doc any
		if jsonErr := json.Unmarshal(data, &doc); jsonErr != nil {
			return fmt.Errorf("parsing embedded schema %s: %w", path, jsonErr)
		}
		return c.AddResource(path, doc)
	})
	if err != nil {
		return nil, fmt.Errorf("registering embedded schemas: %w", err)
	}
	return c, nil
}
