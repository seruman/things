package pkg2

import "testing"

func TestSome(t *testing.T) {
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
