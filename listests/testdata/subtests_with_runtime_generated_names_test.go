//go:build testdata

package testdata

import (
	"fmt"
	"testing"
)

func TestWithSubtestsWithRuntimeGeneratedNames(t *testing.T) {
	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("sub-test%d", i), func(t *testing.T) {
			// Dynamic subtest
		})
	}
}
