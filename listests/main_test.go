package main

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSimpleTest(t *testing.T) {
	testFile := "./testdata/simple_test.go"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	got := findTestsInFile(file, fset, testFile, "testdata")

	expected := []*TestInfo{
		{
			Name:             "TestSimple",
			FullName:         "TestSimple",
			PackageName:      "testdata",
			FileName:         testFile,
			HasGeneratedName: false,
			IsSubtest:        false,
			Line:             7,
			Column:           6,
		},
	}

	if diff := cmp.Diff(expected, got, testInfoCmpOpts()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestWithSubtests(t *testing.T) {
	testFile := "./testdata/with_subtests_test.go"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	got := findTestsInFile(file, fset, testFile, "testdata")

	expected := []*TestInfo{
		{
			Name:             "TestWithSubtests",
			FullName:         "TestWithSubtests",
			PackageName:      "testdata",
			FileName:         testFile,
			HasGeneratedName: false,
			IsSubtest:        false,
			Line:             7,
			Column:           6,
			SubTests: []*TestInfo{
				{
					Name:             "sub1",
					DisplayName:      "sub1",
					FullName:         "TestWithSubtests/sub1",
					FullDisplayName:  "TestWithSubtests/sub1",
					PackageName:      "testdata",
					FileName:         testFile,
					HasGeneratedName: false,
					IsSubtest:        true,
					Line:             8,
					Column:           2,
				},
				{
					Name:             "sub2",
					DisplayName:      "sub2",
					FullName:         "TestWithSubtests/sub2",
					FullDisplayName:  "TestWithSubtests/sub2",
					PackageName:      "testdata",
					FileName:         testFile,
					HasGeneratedName: false,
					IsSubtest:        true,
					Line:             12,
					Column:           2,
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, testInfoCmpOpts()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestNestedSubtests(t *testing.T) {
	testFile := "./testdata/nested_subtests_test.go"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	got := findTestsInFile(file, fset, testFile, "testdata")

	expected := []*TestInfo{
		{
			Name:             "TestWithNestedSubtests",
			FullName:         "TestWithNestedSubtests",
			PackageName:      "testdata",
			FileName:         testFile,
			HasGeneratedName: false,
			IsSubtest:        false,
			Line:             7,
			Column:           6,
			SubTests: []*TestInfo{
				{
					Name:             "level1",
					DisplayName:      "level1",
					FullName:         "TestWithNestedSubtests/level1",
					FullDisplayName:  "TestWithNestedSubtests/level1",
					PackageName:      "testdata",
					FileName:         testFile,
					HasGeneratedName: false,
					IsSubtest:        true,
					Line:             8,
					Column:           2,
					SubTests: []*TestInfo{
						{
							Name:             "level2",
							DisplayName:      "level2",
							FullName:         "TestWithNestedSubtests/level1/level2",
							FullDisplayName:  "TestWithNestedSubtests/level1/level2",
							PackageName:      "testdata",
							FileName:         testFile,
							HasGeneratedName: false,
							IsSubtest:        true,
							Line:             9,
							Column:           3,
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, testInfoCmpOpts()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestSubtestsWithRuntimeGeneratedNames(t *testing.T) {
	testFile := "./testdata/subtests_with_runtime_generated_names_test.go"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, testFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	got := findTestsInFile(file, fset, testFile, "testdata")

	expected := []*TestInfo{
		{
			Name:             "TestWithSubtestsWithRuntimeGeneratedNames",
			FullName:         "TestWithSubtestsWithRuntimeGeneratedNames",
			PackageName:      "testdata",
			FileName:         testFile,
			HasGeneratedName: false,
			IsSubtest:        false,
			SubTests: []*TestInfo{
				{
					Name:             "<fmt.Sprintf(\"sub-test%d\", i)>",
					FullName:         "TestWithSubtestsWithRuntimeGeneratedNames/<fmt.Sprintf(\"sub-test%d\", i)>",
					PackageName:      "testdata",
					FileName:         testFile,
					HasGeneratedName: true,
					IsSubtest:        true,
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, cmpopts.IgnoreFields(TestInfo{}, "Line", "Column")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func testInfoCmpOpts() cmp.Option {
	return cmp.Options{
		cmpopts.SortSlices(func(a, b *TestInfo) bool {
			return a.FullName < b.FullName
		}),
	}
}
