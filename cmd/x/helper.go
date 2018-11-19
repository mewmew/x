package main

import (
	"github.com/mewkiz/pkg/jsonutil"
	"github.com/mewkiz/pkg/osutil"
)

// parseJSON parses the given JSON file and stores the result into v.
func parseJSON(jsonPath string, v interface{}) error {
	if !osutil.Exists(jsonPath) {
		warn.Printf("unable to locate JSON file %q", jsonPath)
		return nil
	}
	dbg.Printf("parseJson(jsonPath = %q, v = %T)", jsonPath, v)
	return jsonutil.ParseFile(jsonPath, v)
}
