package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
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
			Package:          "testdata",
			File:             testFile,
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
			Package:          "testdata",
			File:             testFile,
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
					Package:          "testdata",
					File:             testFile,
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
					Package:          "testdata",
					File:             testFile,
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
			Package:          "testdata",
			File:             testFile,
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
					Package:          "testdata",
					File:             testFile,
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
							Package:          "testdata",
							File:             testFile,
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
			Package:          "testdata",
			File:             testFile,
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
					Package:          "testdata",
					File:             testFile,
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

func TestSubPackages(t *testing.T) {
	logger := func(string, ...any) {
	}
	got, err := findTestsInPackages(
		t.Context(),
		[]string{"./testdata/subpkg/pkg1", "./testdata/subpkg/pkg2"},
		nil,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to find tests in subpackages: %v", err)
	}

	absPath := func(t *testing.T, path string) string {
		abs, err := filepath.Abs(path)
		if err != nil {
			t.Fatalf("Failed to get absolute path for %s: %v", path, err)
		}
		return abs
	}

	expected := []*TestInfo{
		{
			Name:     "TestSome",
			FullName: "TestSome",
			Package:  "listests/testdata/subpkg/pkg1",
			File:     absPath(t, "./testdata/subpkg/pkg1/some_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 5, Column: 6},
				End:   SourcePosition{Line: 25, Column: 2},
			},
			SubTests: []*TestInfo{
				{
					Name:            "sub-test1",
					DisplayName:     "sub-test1",
					FullName:        "TestSome/sub-test1",
					FullDisplayName: "TestSome/sub-test1",
					Package:         "listests/testdata/subpkg/pkg1",
					File:            absPath(t, "./testdata/subpkg/pkg1/some_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 8, Column: 2},
						End:   SourcePosition{Line: 24, Column: 4},
					},
					IsSubtest: true,
					SubTests: []*TestInfo{
						{
							Name:            "sub-sub-test1",
							DisplayName:     "sub-sub-test1",
							FullName:        "TestSome/sub-test1/sub-sub-test1",
							FullDisplayName: "TestSome/sub-test1/sub-sub-test1",
							Package:         "listests/testdata/subpkg/pkg1",
							File:            absPath(t, "./testdata/subpkg/pkg1/some_test.go"),
							Range: SourceRange{
								Start: SourcePosition{Line: 11, Column: 3},
								End:   SourcePosition{Line: 13, Column: 5},
							},
							IsSubtest: true,
						},
						{
							Name:            "sub-sub-test2",
							DisplayName:     "sub-sub-test2",
							FullName:        "TestSome/sub-test1/sub-sub-test2",
							FullDisplayName: "TestSome/sub-test1/sub-sub-test2",
							Package:         "listests/testdata/subpkg/pkg1",
							File:            absPath(t, "./testdata/subpkg/pkg1/some_test.go"),
							Range: SourceRange{
								Start: SourcePosition{Line: 15, Column: 3},
								End:   SourcePosition{Line: 23, Column: 5},
							},
							IsSubtest: true,
							SubTests: []*TestInfo{
								// sub-sub-sub-test1
								// sub-sub-sub-test2
								{
									Name:            "sub-sub-sub-test1",
									DisplayName:     "sub-sub-sub-test1",
									FullName:        "TestSome/sub-test1/sub-sub-test2/sub-sub-sub-test1",
									FullDisplayName: "TestSome/sub-test1/sub-sub-test2/sub-sub-sub-test1",
									Package:         "listests/testdata/subpkg/pkg1",
									File:            absPath(t, "./testdata/subpkg/pkg1/some_test.go"),
									Range: SourceRange{
										Start: SourcePosition{Line: 16, Column: 4},
										End:   SourcePosition{Line: 18, Column: 6},
									},
									IsSubtest: true,
								},
								{
									Name:            "sub-sub-sub-test2",
									DisplayName:     "sub-sub-sub-test2",
									FullName:        "TestSome/sub-test1/sub-sub-test2/sub-sub-sub-test2",
									FullDisplayName: "TestSome/sub-test1/sub-sub-test2/sub-sub-sub-test2",
									Package:         "listests/testdata/subpkg/pkg1",
									File:            absPath(t, "./testdata/subpkg/pkg1/some_test.go"),
									Range: SourceRange{
										Start: SourcePosition{Line: 20, Column: 4},
										End:   SourcePosition{Line: 22, Column: 6},
									},
									IsSubtest: true,
								},
							},
						},
					},
				},
			},
		},

		{
			Name:     "TestSome",
			FullName: "TestSome",
			Package:  "listests/testdata/subpkg/pkg2",
			File:     absPath(t, "./testdata/subpkg/pkg2/some_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 5, Column: 6},
				End:   SourcePosition{Line: 25, Column: 2},
			},
			SubTests: []*TestInfo{
				{
					Name:            "sub-test1",
					DisplayName:     "sub-test1",
					FullName:        "TestSome/sub-test1",
					FullDisplayName: "TestSome/sub-test1",
					Package:         "listests/testdata/subpkg/pkg2",
					File:            absPath(t, "./testdata/subpkg/pkg2/some_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 8, Column: 2},
						End:   SourcePosition{Line: 24, Column: 4},
					},
					IsSubtest: true,
					SubTests: []*TestInfo{
						{
							Name:            "sub-sub-test1",
							DisplayName:     "sub-sub-test1",
							FullName:        "TestSome/sub-test1/sub-sub-test1",
							FullDisplayName: "TestSome/sub-test1/sub-sub-test1",
							Package:         "listests/testdata/subpkg/pkg2",
							File:            absPath(t, "./testdata/subpkg/pkg2/some_test.go"),
							Range: SourceRange{
								Start: SourcePosition{Line: 11, Column: 3},
								End:   SourcePosition{Line: 13, Column: 5},
							},
							IsSubtest: true,
						},
						{
							Name:            "sub-sub-test2",
							DisplayName:     "sub-sub-test2",
							FullName:        "TestSome/sub-test1/sub-sub-test2",
							FullDisplayName: "TestSome/sub-test1/sub-sub-test2",
							Package:         "listests/testdata/subpkg/pkg2",
							File:            absPath(t, "./testdata/subpkg/pkg2/some_test.go"),
							Range: SourceRange{
								Start: SourcePosition{Line: 15, Column: 3},
								End:   SourcePosition{Line: 23, Column: 5},
							},
							IsSubtest: true,
							SubTests: []*TestInfo{
								// sub-sub-sub-test1
								// sub-sub-sub-test2
								{
									Name:            "sub-sub-sub-test1",
									DisplayName:     "sub-sub-sub-test1",
									FullName:        "TestSome/sub-test1/sub-sub-test2/sub-sub-sub-test1",
									FullDisplayName: "TestSome/sub-test1/sub-sub-test2/sub-sub-sub-test1",
									Package:         "listests/testdata/subpkg/pkg2",
									File:            absPath(t, "./testdata/subpkg/pkg2/some_test.go"),
									Range: SourceRange{
										Start: SourcePosition{Line: 16, Column: 4},
										End:   SourcePosition{Line: 18, Column: 6},
									},
									IsSubtest: true,
								},
								{
									Name:            "sub-sub-sub-test2",
									DisplayName:     "sub-sub-sub-test2",
									FullName:        "TestSome/sub-test1/sub-sub-test2/sub-sub-sub-test2",
									FullDisplayName: "TestSome/sub-test1/sub-sub-test2/sub-sub-sub-test2",
									Package:         "listests/testdata/subpkg/pkg2",
									File:            absPath(t, "./testdata/subpkg/pkg2/some_test.go"),
									Range: SourceRange{
										Start: SourcePosition{Line: 20, Column: 4},
										End:   SourcePosition{Line: 22, Column: 6},
									},
									IsSubtest: true,
								},
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
