package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFindTestsInPackages(t *testing.T) {
	modfs := os.DirFS("./internal/testmodule")

	dir := t.TempDir()
	absPath := func(path string) string {
		return filepath.Join(dir, path)
	}

	if err := os.CopyFS(dir, modfs); err != nil {
		t.Fatalf("copy-fs: %v", err)
	}

	logfn := func(string, ...any) {}
	got, err := findTestsInPackages(
		t.Context(),
		dir,
		[]string{"./..."},
		nil,
		logfn,
	)
	if err != nil {
		t.Fatalf("find-tests-in-packages: %v", err)
	}

	want := []*TestInfo{
		{
			Name:      "TestSimple",
			FullName:  "TestSimple",
			Package:   "testmodule",
			Directory: dir,
			File:      absPath("/some_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 8, Column: 6},
				End:   SourcePosition{Line: 10, Column: 2},
			},
		},
		{
			Name:      "TestSubTests",
			FullName:  "TestSubTests",
			Package:   "testmodule",
			Directory: dir,
			File:      absPath("/some_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 12, Column: 6},
				End:   SourcePosition{Line: 19, Column: 2},
			},
			SubTests: []*TestInfo{
				{
					Name:            "t1",
					DisplayName:     "t1",
					FullName:        "TestSubTests/t1",
					FullDisplayName: "TestSubTests/t1",
					Package:         "testmodule",
					Directory:       dir,
					File:            absPath("/some_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 13, Column: 2},
						End:   SourcePosition{Line: 15, Column: 4},
					},
					IsSubtest: true,
				},
				{
					Name:            "t2",
					DisplayName:     "t2",
					FullName:        "TestSubTests/t2",
					FullDisplayName: "TestSubTests/t2",
					Package:         "testmodule",
					Directory:       dir,
					File:            absPath("/some_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 16, Column: 2},
						End:   SourcePosition{Line: 18, Column: 4},
					},
					IsSubtest: true,
				},
			},
		},
		{
			Name:      "TestNestedSubTests",
			FullName:  "TestNestedSubTests",
			Package:   "testmodule",
			Directory: dir,
			File:      absPath("/some_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 21, Column: 6},
				End:   SourcePosition{Line: 27, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:            "t1",
				DisplayName:     "t1",
				FullName:        "TestNestedSubTests/t1",
				FullDisplayName: "TestNestedSubTests/t1",
				Package:         "testmodule",
				Directory:       dir,
				File:            absPath("/some_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 22, Column: 2},
					End:   SourcePosition{Line: 26, Column: 4},
				},
				IsSubtest: true,
				SubTests: []*TestInfo{{
					Name:            "t1",
					DisplayName:     "t1",
					FullName:        "TestNestedSubTests/t1/t1",
					FullDisplayName: "TestNestedSubTests/t1/t1",
					Package:         "testmodule",
					Directory:       dir,
					File:            absPath("/some_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 23, Column: 3},
						End:   SourcePosition{Line: 25, Column: 5},
					},
					IsSubtest: true,
				}},
			}},
		},
		{
			Name:      "TestSubTestsWithGeneratedNames",
			FullName:  "TestSubTestsWithGeneratedNames",
			Package:   "testmodule",
			Directory: dir,
			File:      absPath("/some_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 29, Column: 6},
				End:   SourcePosition{Line: 35, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:      `<fmt.Sprintf("t%v", i)>`,
				FullName:  `TestSubTestsWithGeneratedNames/<fmt.Sprintf("t%v", i)>`,
				Package:   "testmodule",
				Directory: dir,
				File:      absPath("/some_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 31, Column: 3},
					End:   SourcePosition{Line: 33, Column: 5},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}},
		},
		{
			Name:      "TestTable",
			FullName:  "TestTable",
			Package:   "testmodule",
			Directory: dir,
			File:      absPath("/some_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 37, Column: 6},
				End:   SourcePosition{Line: 52, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:      "<c.name>",
				FullName:  "TestTable/<c.name>",
				Package:   "testmodule",
				Directory: dir,
				File:      absPath("/some_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 48, Column: 3},
					End:   SourcePosition{Line: 50, Column: 5},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}},
		},
		{
			Name:      "TestSimple",
			FullName:  "TestSimple",
			Package:   "subpkg",
			Directory: absPath("/subpkg"),
			File:      absPath("subpkg/subpkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 8, Column: 6},
				End:   SourcePosition{Line: 10, Column: 2},
			},
		},
		{
			Name:      "TestSubTests",
			FullName:  "TestSubTests",
			Package:   "subpkg",
			Directory: absPath("/subpkg"),
			File:      absPath("subpkg/subpkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 12, Column: 6},
				End:   SourcePosition{Line: 19, Column: 2},
			},
			SubTests: []*TestInfo{
				{
					Name:            "t1",
					DisplayName:     "t1",
					FullName:        "TestSubTests/t1",
					FullDisplayName: "TestSubTests/t1",
					Package:         "subpkg",
					Directory:       absPath("/subpkg"),
					File:            absPath("subpkg/subpkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 13, Column: 2},
						End:   SourcePosition{Line: 15, Column: 4},
					},
					IsSubtest: true,
				},
				{
					Name:            "t2",
					DisplayName:     "t2",
					FullName:        "TestSubTests/t2",
					FullDisplayName: "TestSubTests/t2",
					Package:         "subpkg",
					Directory:       absPath("/subpkg"),
					File:            absPath("subpkg/subpkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 16, Column: 2},
						End:   SourcePosition{Line: 18, Column: 4},
					},
					IsSubtest: true,
				},
			},
		},
		{
			Name:      "TestNestedSubTests",
			FullName:  "TestNestedSubTests",
			Package:   "subpkg",
			Directory: absPath("/subpkg"),
			File:      absPath("subpkg/subpkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 21, Column: 6},
				End:   SourcePosition{Line: 27, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:            "t1",
				DisplayName:     "t1",
				FullName:        "TestNestedSubTests/t1",
				FullDisplayName: "TestNestedSubTests/t1",
				Package:         "subpkg",
				Directory:       absPath("/subpkg"),
				File:            absPath("subpkg/subpkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 22, Column: 2},
					End:   SourcePosition{Line: 26, Column: 4},
				},
				IsSubtest: true,
				SubTests: []*TestInfo{{
					Name:            "t1",
					DisplayName:     "t1",
					FullName:        "TestNestedSubTests/t1/t1",
					FullDisplayName: "TestNestedSubTests/t1/t1",
					Package:         "subpkg",
					Directory:       absPath("/subpkg"),
					File:            absPath("subpkg/subpkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 23, Column: 3},
						End:   SourcePosition{Line: 25, Column: 5},
					},
					IsSubtest: true,
				}},
			}},
		},
		{
			Name:      "TestSubTestsWithGeneratedNames",
			FullName:  "TestSubTestsWithGeneratedNames",
			Package:   "subpkg",
			Directory: absPath("/subpkg"),
			File:      absPath("subpkg/subpkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 29, Column: 6},
				End:   SourcePosition{Line: 35, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:      `<fmt.Sprintf("t%v", i)>`,
				FullName:  `TestSubTestsWithGeneratedNames/<fmt.Sprintf("t%v", i)>`,
				Package:   "subpkg",
				Directory: absPath("/subpkg"),
				File:      absPath("subpkg/subpkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 31, Column: 3},
					End:   SourcePosition{Line: 33, Column: 5},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}},
		},
		{
			Name:      "TestTable",
			FullName:  "TestTable",
			Package:   "subpkg",
			Directory: absPath("/subpkg"),
			File:      absPath("subpkg/subpkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 37, Column: 6},
				End:   SourcePosition{Line: 52, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:      "<c.name>",
				FullName:  "TestTable/<c.name>",
				Package:   "subpkg",
				Directory: absPath("/subpkg"),
				File:      absPath("subpkg/subpkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 48, Column: 3},
					End:   SourcePosition{Line: 50, Column: 5},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}},
		},
		{
			Name:      "TestSimple",
			FullName:  "TestSimple",
			Package:   "subpkg/pkg1",
			Directory: absPath("/subpkg/pkg1"),
			File:      absPath("/subpkg/pkg1/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 8, Column: 6},
				End:   SourcePosition{Line: 10, Column: 2},
			},
		},
		{
			Name:      "TestSubTests",
			FullName:  "TestSubTests",
			Package:   "subpkg/pkg1",
			Directory: absPath("/subpkg/pkg1"),
			File:      absPath("/subpkg/pkg1/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 12, Column: 6},
				End:   SourcePosition{Line: 19, Column: 2},
			},
			SubTests: []*TestInfo{
				{
					Name:            "t1",
					DisplayName:     "t1",
					FullName:        "TestSubTests/t1",
					FullDisplayName: "TestSubTests/t1",
					Package:         "subpkg/pkg1",
					Directory:       absPath("/subpkg/pkg1"),
					File:            absPath("/subpkg/pkg1/pkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 13, Column: 2},
						End:   SourcePosition{Line: 15, Column: 4},
					},
					IsSubtest: true,
				},
				{
					Name:            "t2",
					DisplayName:     "t2",
					FullName:        "TestSubTests/t2",
					FullDisplayName: "TestSubTests/t2",
					Package:         "subpkg/pkg1",
					Directory:       absPath("/subpkg/pkg1"),
					File:            absPath("/subpkg/pkg1/pkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 16, Column: 2},
						End:   SourcePosition{Line: 18, Column: 4},
					},
					IsSubtest: true,
				},
			},
		},
		{
			Name:      "TestNestedSubTests",
			FullName:  "TestNestedSubTests",
			Package:   "subpkg/pkg1",
			Directory: absPath("/subpkg/pkg1"),
			File:      absPath("/subpkg/pkg1/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 21, Column: 6},
				End:   SourcePosition{Line: 27, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:            "t1",
				DisplayName:     "t1",
				FullName:        "TestNestedSubTests/t1",
				FullDisplayName: "TestNestedSubTests/t1",
				Package:         "subpkg/pkg1",
				Directory:       absPath("/subpkg/pkg1"),
				File:            absPath("/subpkg/pkg1/pkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 22, Column: 2},
					End:   SourcePosition{Line: 26, Column: 4},
				},
				IsSubtest: true,
				SubTests: []*TestInfo{{
					Name:            "t1",
					DisplayName:     "t1",
					FullName:        "TestNestedSubTests/t1/t1",
					FullDisplayName: "TestNestedSubTests/t1/t1",
					Package:         "subpkg/pkg1",
					Directory:       absPath("/subpkg/pkg1"),
					File:            absPath("/subpkg/pkg1/pkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 23, Column: 3},
						End:   SourcePosition{Line: 25, Column: 5},
					},
					IsSubtest: true,
				}},
			}},
		},
		{
			Name:      "TestSubTestsWithGeneratedNames",
			FullName:  "TestSubTestsWithGeneratedNames",
			Package:   "subpkg/pkg1",
			Directory: absPath("/subpkg/pkg1"),
			File:      absPath("/subpkg/pkg1/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 29, Column: 6},
				End:   SourcePosition{Line: 35, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:      `<fmt.Sprintf("t%v", i)>`,
				FullName:  `TestSubTestsWithGeneratedNames/<fmt.Sprintf("t%v", i)>`,
				Package:   "subpkg/pkg1",
				Directory: absPath("/subpkg/pkg1"),
				File:      absPath("/subpkg/pkg1/pkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 31, Column: 3},
					End:   SourcePosition{Line: 33, Column: 5},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}},
		},
		{
			Name:      "TestTable",
			FullName:  "TestTable",
			Package:   "subpkg/pkg1",
			Directory: absPath("/subpkg/pkg1"),
			File:      absPath("/subpkg/pkg1/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 37, Column: 6},
				End:   SourcePosition{Line: 52, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:      "<c.name>",
				FullName:  "TestTable/<c.name>",
				Package:   "subpkg/pkg1",
				Directory: absPath("/subpkg/pkg1"),
				File:      absPath("/subpkg/pkg1/pkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 48, Column: 3},
					End:   SourcePosition{Line: 50, Column: 5},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}},
		},
		{
			Name:      "TestSimple",
			FullName:  "TestSimple",
			Package:   "subpkg/pkg2",
			Directory: absPath("/subpkg/pkg2"),
			File:      absPath("/subpkg/pkg2/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 8, Column: 6},
				End:   SourcePosition{Line: 10, Column: 2},
			},
		},
		{
			Name:      "TestSubTests",
			FullName:  "TestSubTests",
			Package:   "subpkg/pkg2",
			Directory: absPath("/subpkg/pkg2"),
			File:      absPath("/subpkg/pkg2/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 12, Column: 6},
				End:   SourcePosition{Line: 19, Column: 2},
			},
			SubTests: []*TestInfo{
				{
					Name:            "t1",
					DisplayName:     "t1",
					FullName:        "TestSubTests/t1",
					FullDisplayName: "TestSubTests/t1",
					Package:         "subpkg/pkg2",
					Directory:       absPath("/subpkg/pkg2"),
					File:            absPath("/subpkg/pkg2/pkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 13, Column: 2},
						End:   SourcePosition{Line: 15, Column: 4},
					},
					IsSubtest: true,
				},
				{
					Name:            "t2",
					DisplayName:     "t2",
					FullName:        "TestSubTests/t2",
					FullDisplayName: "TestSubTests/t2",
					Package:         "subpkg/pkg2",
					Directory:       absPath("/subpkg/pkg2"),
					File:            absPath("/subpkg/pkg2/pkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 16, Column: 2},
						End:   SourcePosition{Line: 18, Column: 4},
					},
					IsSubtest: true,
				},
			},
		},
		{
			Name:      "TestNestedSubTests",
			FullName:  "TestNestedSubTests",
			Package:   "subpkg/pkg2",
			Directory: absPath("/subpkg/pkg2"),
			File:      absPath("/subpkg/pkg2/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 21, Column: 6},
				End:   SourcePosition{Line: 27, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:            "t1",
				DisplayName:     "t1",
				FullName:        "TestNestedSubTests/t1",
				FullDisplayName: "TestNestedSubTests/t1",
				Package:         "subpkg/pkg2",
				Directory:       absPath("/subpkg/pkg2"),
				File:            absPath("/subpkg/pkg2/pkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 22, Column: 2},
					End:   SourcePosition{Line: 26, Column: 4},
				},
				IsSubtest: true,
				SubTests: []*TestInfo{{
					Name:            "t1",
					DisplayName:     "t1",
					FullName:        "TestNestedSubTests/t1/t1",
					FullDisplayName: "TestNestedSubTests/t1/t1",
					Package:         "subpkg/pkg2",
					Directory:       absPath("/subpkg/pkg2"),
					File:            absPath("/subpkg/pkg2/pkg_test.go"),
					Range: SourceRange{
						Start: SourcePosition{Line: 23, Column: 3},
						End:   SourcePosition{Line: 25, Column: 5},
					},
					IsSubtest: true,
				}},
			}},
		},
		{
			Name:      "TestSubTestsWithGeneratedNames",
			FullName:  "TestSubTestsWithGeneratedNames",
			Package:   "subpkg/pkg2",
			Directory: absPath("/subpkg/pkg2"),
			File:      absPath("/subpkg/pkg2/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 29, Column: 6},
				End:   SourcePosition{Line: 35, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:      `<fmt.Sprintf("t%v", i)>`,
				FullName:  `TestSubTestsWithGeneratedNames/<fmt.Sprintf("t%v", i)>`,
				Package:   "subpkg/pkg2",
				Directory: absPath("/subpkg/pkg2"),
				File:      absPath("/subpkg/pkg2/pkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 31, Column: 3},
					End:   SourcePosition{Line: 33, Column: 5},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}},
		},
		{
			Name:      "TestTable",
			FullName:  "TestTable",
			Package:   "subpkg/pkg2",
			Directory: absPath("/subpkg/pkg2"),
			File:      absPath("/subpkg/pkg2/pkg_test.go"),
			Range: SourceRange{
				Start: SourcePosition{Line: 37, Column: 6},
				End:   SourcePosition{Line: 52, Column: 2},
			},
			SubTests: []*TestInfo{{
				Name:      "<c.name>",
				FullName:  "TestTable/<c.name>",
				Package:   "subpkg/pkg2",
				Directory: absPath("/subpkg/pkg2"),
				File:      absPath("/subpkg/pkg2/pkg_test.go"),
				Range: SourceRange{
					Start: SourcePosition{Line: 48, Column: 3},
					End:   SourcePosition{Line: 50, Column: 5},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}},
		},
	}
	if diff := cmp.Diff(want, got, testInfoCmpOpts()); diff != "" {
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
