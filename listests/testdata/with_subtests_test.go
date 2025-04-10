//go:build testdata

package testdata

import "testing"

func TestWithSubtests(t *testing.T) {
	t.Run("sub1", func(t *testing.T) {
		// Subtest 1
	})

	t.Run("sub2", func(t *testing.T) {
		// Subtest 2
	})
}
