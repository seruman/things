//go:build integration

package testdata

import (
	"testing"
)

func TestTableTest(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "test1"},
		{name: "test2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test: %s", tt.name)
		})
	}
}
