// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"bytes"
	"encoding/csv"
	"testing"
)

func TestWriteCSV(t *testing.T) {
	g := testGraph()
	var buf bytes.Buffer
	if err := WriteCSV(&buf, g); err != nil {
		t.Fatal(err)
	}

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Header + 3 data rows (account, email, domain).
	if len(records) < 4 {
		t.Errorf("expected at least 4 rows (header + 3 nodes), got %d", len(records))
	}

	// Check header.
	header := records[0]
	expected := []string{"id", "type", "label", "source_module", "confidence", "wave"}
	for i, col := range expected {
		if i >= len(header) || header[i] != col {
			t.Errorf("header[%d] = %q, want %q", i, header[i], col)
		}
	}
}
