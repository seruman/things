//go:build testdata

package testdata

import "testing"

func TestWithNestedSubtests(t *testing.T) {
	t.Run("level1", func(t *testing.T) {
		t.Run("level2", func(t *testing.T) {
			// Nested subtest
		})
	})
}
