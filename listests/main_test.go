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
			Range: SourceRange{
				Start: SourcePosition{7, 6},
				End:   SourcePosition{9, 2},
			},
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
			Range: SourceRange{
				Start: SourcePosition{7, 6},
				End:   SourcePosition{15, 2},
			},
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
					Range: SourceRange{
						Start: SourcePosition{8, 2},
						End:   SourcePosition{10, 4},
					},
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
					Range: SourceRange{
						Start: SourcePosition{12, 2},
						End:   SourcePosition{14, 4},
					},
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
			Range: SourceRange{
				Start: SourcePosition{7, 6},
				End:   SourcePosition{13, 2},
			},
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
					Range: SourceRange{
						Start: SourcePosition{8, 2},
						End:   SourcePosition{12, 4},
					},
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
							Range: SourceRange{
								Start: SourcePosition{9, 3},
								End:   SourcePosition{11, 5},
							},
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
			Range: SourceRange{
				Start: SourcePosition{10, 6},
				End:   SourcePosition{16, 2},
			},
			SubTests: []*TestInfo{
				{
					Name:             "<fmt.Sprintf(\"sub-test%d\", i)>",
					FullName:         "TestWithSubtestsWithRuntimeGeneratedNames/<fmt.Sprintf(\"sub-test%d\", i)>",
					PackageName:      "testdata",
					FileName:         testFile,
					HasGeneratedName: true,
					IsSubtest:        true,
					Range: SourceRange{
						Start: SourcePosition{12, 3},
						End:   SourcePosition{14, 5},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
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
