package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
	"iter"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/tools/go/packages"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := realmain(
		ctx,
		os.Stdin,
		os.Stdout,
		os.Stderr,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realmain(
	ctx context.Context,
	_ io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	osargs []string,
) error {
	fs := flag.NewFlagSet("listests", flag.ExitOnError)
	fs.SetOutput(stderr)

	var (
		flagTags    string
		flagVerbose bool
		flagVimgrep bool
		flagFormat  string
	)

	fs.StringVar(&flagTags, "tags", "", "comma-separated list of build tags to apply")
	fs.BoolVar(&flagVerbose, "v", false, "verbose mode")
	fs.BoolVar(&flagVimgrep, "vimgrep", false, "output in ripgrep's vimgrep format")
	fs.StringVar(&flagFormat, "format", "", "output format")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [options] [packages...]\n", fs.Name())
		fmt.Fprintf(fs.Output(), "If no arguments are provided, ./... is used.\n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(osargs[1:]); err != nil {
		return err
	}

	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./."}
	}

	var buildTags []string
	if flagTags != "" {
		buildTags = strings.Split(flagTags, ",")
	}

	// TODO: slog
	logger := func(format string, args ...any) {
		if flagVerbose {
			fmt.Fprintf(stderr, format, args...)
		}
	}

	logger("Discovering tests...\n")
	tests, err := findTestsInPackages(
		ctx,
		".",
		patterns,
		buildTags,
		logger,
	)
	if err != nil {
		return err
	}

	slices.SortStableFunc(tests, testInfoCmp)

	if flagVimgrep {
		if flagFormat != "" {
			return fmt.Errorf("cannot use -vimgrep and -format together")
		}

		flagFormat = "{{.RelativeFileName}}:{{.Range.Start.Line}}:{{.Range.Start.Column}}:{{.PackageName}}:{{.FullName}}"
	}

	if flagFormat == "" {
		flagFormat = "{{.FullName}}"
	}

	tmpl, err := template.New("format").Parse(flagFormat)
	if err != nil {
		return fmt.Errorf("failed to parse format: %w", err)
	}

	for test := range iterTests(tests) {
		cwd, _ := os.Getwd()
		relativePath, err := filepath.Rel(cwd, test.File)
		if err != nil {
			return fmt.Errorf("failed to get relative file path: %w", err)
		}

		relativeDir, err := filepath.Rel(cwd, test.Directory)
		if err != nil {
			return fmt.Errorf("failed to get relative directory: %w", err)
		}
		templateData := struct {
			TestInfo
			RelativeFileName  string
			RelativeDirectory string
		}{
			TestInfo:          *test,
			RelativeFileName:  relativePath,
			RelativeDirectory: relativeDir,
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, templateData); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		if _, err := fmt.Fprintln(stdout, buf.String()); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}

	}

	return nil
}

type TestInfo struct {
	// Name of this test (not including parents)
	Name string `json:"name"`

	// Display name (go test output)
	DisplayName string `json:"displayName"`

	// Full path including parent tests
	FullName string `json:"fullName"`

	// Full display name (go test output)
	FullDisplayName string `json:"fullDisplayName"`

	// Package name
	Package string `json:"package"`

	// Directory where the test file is located
	Directory string `json:"directory"`

	// File where the test is defined
	File string `json:"file"`

	// Source code location
	Range SourceRange `json:"range"`

	// Whether the test name was runtime generated
	HasGeneratedName bool `json:"hasGeneratedName"`

	// Whether it's a subtest or top-level
	IsSubtest bool `json:"isSubtest"`

	// Subtests
	SubTests []*TestInfo `json:"subTests,omitzero"`
}

type SourceRange struct {
	Start SourcePosition
	End   SourcePosition
}

type SourcePosition struct {
	Line   int
	Column int
}

func findTestsInPackages(
	ctx context.Context,
	directory string,
	patterns []string,
	buildTags []string,
	logger func(string, ...any),
) ([]*TestInfo, error) {
	buildFlags := []string{}
	if len(buildTags) > 0 {
		buildFlags = append(buildFlags, fmt.Sprintf("-tags=%s", strings.Join(buildTags, " ")))
	}

	cfg := &packages.Config{
		Dir:        directory,
		Mode:       packages.LoadFiles | packages.NeedSyntax | packages.NeedForTest | packages.NeedModule,
		Context:    ctx,
		Tests:      true,
		BuildFlags: buildFlags,
	}

	logger("Loading packages...\n")
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found")
	}

	var allTests []*TestInfo
	for _, pkg := range pkgs {
		if pkg.ForTest == "" {
			continue
		}

		// TODO:
		// fmt.Println("Errors:", pkg.Errors)
		// fmt.Println("TypeErrors:", pkg.TypeErrors)

		for _, file := range pkg.Syntax {
			filename := pkg.Fset.Position(file.Pos()).Filename
			if !strings.HasSuffix(filename, "_test.go") {
				continue
			}

			moduleName := pkg.Module.Path
			pkgPath := pkg.PkgPath
			packageName := strings.TrimPrefix(pkgPath, moduleName+"/")
			directory := pkg.Dir

			logger("Processing %s in package %s...\n", filename, packageName)
			tests := findTestsInFile(file, pkg.Fset, filename, packageName, directory)
			allTests = append(allTests, tests...)
		}
	}

	return allTests, nil
}

func findTestsInFile(file *ast.File, fset *token.FileSet, filename, pkgName string, dir string) []*TestInfo {
	var tests []*TestInfo

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil {
			continue
		}

		if strings.HasPrefix(funcDecl.Name.Name, "Test") {
			if isTestFunction(funcDecl) {
				testName := funcDecl.Name.Name

				start := fset.Position(funcDecl.Name.Pos())
				end := fset.Position(funcDecl.End())

				test := &TestInfo{
					Name:      testName,
					FullName:  testName,
					Package:   pkgName,
					Directory: dir,
					File:      filename,
					// Start:            start.Line,
					// Column:           start.Column,
					Range: SourceRange{
						Start: SourcePosition{
							Line:   start.Line,
							Column: start.Column,
						},
						End: SourcePosition{
							Line:   end.Line,
							Column: end.Column,
						},
					},
					HasGeneratedName: false,
					IsSubtest:        false,
					SubTests:         nil,
				}

				if funcDecl.Body != nil {
					findSubtests(funcDecl.Body, test, fset, filename, pkgName)
				}

				tests = append(tests, test)
			}
		}
	}

	return tests
}

// Can do better by checking the parameter type but this is faster and good
// enough. This is how `go test` does it;
// https://github.com/golang/go/blob/2c35900fe4256d6de132cbee6f5a15b29013aac9/src/cmd/go/internal/load/test.go#L766-L779
func isTestFunction(fn *ast.FuncDecl) bool {
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 ||
		fn.Type.Params.List == nil ||
		len(fn.Type.Params.List) != 1 ||
		len(fn.Type.Params.List[0].Names) > 1 {
		return false
	}

	ptr, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	if name, ok := ptr.X.(*ast.Ident); ok && name.Name == "T" {
		return true
	}
	if sel, ok := ptr.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "T" {
		return true
	}

	return false
}

func findSubtests(block *ast.BlockStmt, parentTest *TestInfo, fset *token.FileSet, filename, pkgName string) {
	ast.Inspect(block, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if selExpr.Sel.Name != "Run" {
			return true
		}

		// NOTE: no type check here.
		if len(callExpr.Args) < 2 {
			return true
		}

		start := fset.Position(callExpr.Pos())
		end := fset.Position(callExpr.End())

		var subTest *TestInfo
		switch arg := callExpr.Args[0].(type) {
		case *ast.BasicLit:
			if arg.Kind == token.STRING {
				subtestName := strings.Trim(arg.Value, "\"'`")
				sanitizedSubtestName := rewriteSubTestName(subtestName)

				fullName := parentTest.FullName + "/" + subtestName
				sanitizedFullName := parentTest.FullName + "/" + sanitizedSubtestName

				subTest = &TestInfo{
					Name:            subtestName,
					DisplayName:     sanitizedSubtestName,
					FullName:        fullName,
					FullDisplayName: sanitizedFullName,
					Package:         pkgName,
					Directory:       parentTest.Directory,
					File:            filename,
					Range: SourceRange{
						Start: SourcePosition{
							Line:   start.Line,
							Column: start.Column,
						},
						End: SourcePosition{
							Line:   end.Line,
							Column: end.Column,
						},
					},
					HasGeneratedName: false,
					IsSubtest:        true,
				}
			}
		default:
			// TODO: how to report runtime generated names?
			var buf bytes.Buffer
			err := printer.Fprint(&buf, fset, arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error printing argument: %v\n", err)
				return true
			}
			subtestName := fmt.Sprintf("<%s>", strings.TrimSpace(buf.String()))
			fullName := fmt.Sprintf("%s/%s", parentTest.FullName, subtestName)
			subTest = &TestInfo{
				Name:      subtestName,
				FullName:  fullName,
				Package:   pkgName,
				Directory: parentTest.Directory,
				File:      filename,
				Range: SourceRange{
					Start: SourcePosition{
						Line:   start.Line,
						Column: start.Column,
					},
					End: SourcePosition{
						Line:   end.Line,
						Column: end.Column,
					},
				},
				HasGeneratedName: true,
				IsSubtest:        true,
			}
		}

		if subTest != nil {
			parentTest.SubTests = append(parentTest.SubTests, subTest)

			if funcLit, ok := callExpr.Args[1].(*ast.FuncLit); ok && funcLit.Body != nil {
				findSubtests(funcLit.Body, subTest, fset, filename, pkgName)
			}

			return false
		}

		return true
	})
}

func iterTests(tests []*TestInfo) iter.Seq[*TestInfo] {
	return func(yield func(*TestInfo) bool) {
		// TODO: mutates
		slices.SortStableFunc(tests, testInfoCmp)

		for _, test := range tests {
			if !yield(test) {
				return
			}
			if len(test.SubTests) > 0 {
				iterTests(test.SubTests)(yield)
			}
		}
	}
}

// https://github.com/golang/go/blob/master/src/testing/match.go#L282-L298
func rewriteSubTestName(s string) string {
	b := []byte{}
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			b = append(b, '_')
		case !strconv.IsPrint(r):
			s := strconv.QuoteRune(r)
			b = append(b, s[1:len(s)-1]...)
		default:
			b = append(b, string(r)...)
		}
	}
	return string(b)
}

func testInfoCmp(a, b *TestInfo) int {
	if a.Package != b.Package {
		return strings.Compare(a.Package, b.Package)
	}

	if a.File != b.File {
		return strings.Compare(a.FullName, b.FullName)
	}

	if a.Range.Start.Line != b.Range.Start.Line {
		return a.Range.Start.Line - b.Range.Start.Line
	}

	if a.Range.Start.Column != b.Range.Start.Column {
		return a.Range.Start.Column - b.Range.Start.Column
	}
	return 0
}
