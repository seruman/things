package pkg1

import (
	"fmt"
	"testing"
)

func TestSimple(t *testing.T) {
	t.Skip()
}

func TestSubTests(t *testing.T) {
	t.Run("t1", func(t *testing.T) {
		t.Skip()
	})
	t.Run("t2", func(t *testing.T) {
		t.Skip()
	})
}

func TestNestedSubTests(t *testing.T) {
	t.Run("t1", func(t *testing.T) {
		t.Run("t1", func(t *testing.T) {
			t.Skip()
		})
	})
}

func TestSubTestsWithGeneratedNames(t *testing.T) {
	for i := range 3 {
		t.Run(fmt.Sprintf("t%v", i), func(t *testing.T) {
			t.Skip()
		})
	}
}

func TestTable(t *testing.T) {
	cases := []struct {
		name string
		got  any
		want any
	}{
		{name: "t1", got: "got1", want: "want1"},
		{name: "t2", got: "got2", want: "want2"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Skip()
		})
	}
}

func TestTableTestWithinSubTest(t *testing.T) {
	t.Run("s1", func(t *testing.T) {
		cases := []struct {
			name string
			got  any
			want any
		}{
			{name: "t1", got: "got1", want: "want1"},
			{name: "t2", got: "got2", want: "want2"},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Skip()
			})
		}
	})

	t.Run("s2", func(t *testing.T) {
		cases := []struct {
			name string
			got  any
			want any
		}{
			{name: "t1", got: "got1", want: "want1"},
			{name: "t2", got: "got2", want: "want2"},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Skip()
			})
		}
	})
}

func TestTableTestsWithinSubTestsWithPositionals(t *testing.T) {
	t.Run("t1", func(t *testing.T) {
		cases := []struct {
			input string
			want  any
		}{
			{"tt1", "tt1"},
			{"tt2 with space", "tt2 with space"},
		}
		for _, c := range cases {
			t.Run(c.input, func(t *testing.T) {
				t.Skip()
			})
		}
	})

	t.Run("t2", func(t *testing.T) {
		cases := []struct {
			name  string
			input string
		}{
			{"tt1", "tt1"},
			{"tt2 with space", "tt2 with space"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Skip()
			})
		}
	})
}
