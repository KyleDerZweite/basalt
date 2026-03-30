// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	g := testGraph()
	var buf bytes.Buffer
	if err := WriteJSON(&buf, g); err != nil {
		t.Fatal(err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := out["nodes"]; !ok {
		t.Error("JSON missing 'nodes'")
	}
	if _, ok := out["edges"]; !ok {
		t.Error("JSON missing 'edges'")
	}
}
