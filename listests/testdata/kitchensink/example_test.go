//go:build testdata

package example

import (
	"fmt"
	"testing"
)

func TestTopLevel1(t *testing.T) {
	t.Parallel()

	t.Run("sub-test1", func(t *testing.T) {
		t.Log("Running some-test")

		t.Run("sub-sub-test1", func(t *testing.T) {
			t.Log("Running some-sub-test")
		})

		t.Run("sub-sub-test2", func(t *testing.T) {
			t.Run("sub-sub-sub-test1", func(t *testing.T) {
				t.Log("Running some-sub-sub-test")
			})

			t.Run("sub-sub-sub-test2", func(t *testing.T) {
				t.Log("Running some-sub-sub-test")
			})
		})
	})
}

func TestTopLevel2(t *testing.T) {
	t.Parallel()

	t.Log("Running some-test")
}

func TestTopLevel3(t *testing.T) {
	t.Parallel()
	for i := range 3 {
		t.Run(fmt.Sprintf("sub-test%v", i), func(t *testing.T) {
			t.Log("Running some-test")
		})
	}
}

func TestTopLevel4(t *testing.T) {
	t.Run("sub test with spaces", func(t *testing.T) {
		t.Log("Running some-test")
	})
}
