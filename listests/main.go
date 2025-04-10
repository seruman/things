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
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
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
	)

	fs.StringVar(&flagTags, "tags", "", "comma-separated list of build tags to apply")
	fs.BoolVar(&flagVerbose, "v", false, "verbose mode")

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
	tests, err := findTestsInPackages(ctx, patterns, buildTags, logger)
	if err != nil {
		return err
	}

	slices.SortStableFunc(tests, testInfoCmp)

	var results []string
	for _, test := range tests {
		testnames := flattenTestTree(test)
		results = append(results, testnames...)
	}

	for _, testName := range results {
		fmt.Fprintln(stdout, testName)
	}

	return nil
}

type TestInfo struct {
	Name             string      // Name of this test (not including parents)
	DisplayName      string      // Display name (go test output)
	FullName         string      // Full path including parent tests
	FullDisplayName  string      // Full display name (go test output)
	PackageName      string      // Package containing the test
	FileName         string      // File where the test is defined
	Line             int         // Line number in the file
	Column           int         // Column position
	HasGeneratedName bool        // Whether the test name was runtime generated
	IsSubtest        bool        // Whether it's a subtest or top-level
	SubTests         []*TestInfo // Subtests
}

func findTestsInPackages(
	ctx context.Context,
	patterns []string,
	buildTags []string,
	logger func(string, ...any),
) ([]*TestInfo, error) {
	buildFlags := []string{}
	if len(buildTags) > 0 {
		buildFlags = append(buildFlags, fmt.Sprintf("-tags=%s", strings.Join(buildTags, " ")))
	}

	cfg := &packages.Config{
		Mode:       packages.LoadFiles | packages.NeedSyntax | packages.NeedForTest,
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

			logger("Processing %s in package %s...\n", filename, pkg.Name)
			tests := findTestsInFile(file, pkg.Fset, filename, pkg.Name)
			allTests = append(allTests, tests...)
		}
	}

	return allTests, nil
}

func findTestsInFile(file *ast.File, fset *token.FileSet, filename, pkgName string) []*TestInfo {
	var tests []*TestInfo

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil {
			continue
		}

		if strings.HasPrefix(funcDecl.Name.Name, "Test") {
			if isTestFunction(funcDecl) {
				testName := funcDecl.Name.Name
				pos := fset.Position(funcDecl.Name.Pos())

				test := &TestInfo{
					Name:             testName,
					FullName:         testName,
					PackageName:      pkgName,
					FileName:         filename,
					Line:             pos.Line,
					Column:           pos.Column,
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

		pos := fset.Position(callExpr.Pos())

		var subTest *TestInfo
		switch arg := callExpr.Args[0].(type) {
		case *ast.BasicLit:
			if arg.Kind == token.STRING {
				subtestName := strings.Trim(arg.Value, "\"'`")
				sanitizedSubtestName := rewriteSubTestName(subtestName)

				fullName := parentTest.FullName + "/" + subtestName
				sanitizedFullName := parentTest.FullName + "/" + sanitizedSubtestName

				subTest = &TestInfo{
					Name:             subtestName,
					DisplayName:      sanitizedSubtestName,
					FullName:         fullName,
					FullDisplayName:  sanitizedFullName,
					PackageName:      pkgName,
					FileName:         filename,
					Line:             pos.Line,
					Column:           pos.Column,
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
				Name:             subtestName,
				FullName:         fullName,
				PackageName:      pkgName,
				FileName:         filename,
				Line:             pos.Line,
				Column:           pos.Column,
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

func flattenTestTree(test *TestInfo) []string {
	var testnames []string
	testnames = append(testnames, test.FullName)

	subtests := test.SubTests
	slices.SortStableFunc(subtests, testInfoCmp)

	for _, subTest := range subtests {
		subtestnames := flattenTestTree(subTest)
		testnames = append(testnames, subtestnames...)
	}

	return testnames
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
	if a.FullName != b.FullName {
		return strings.Compare(a.FullName, b.FullName)
	}
	if a.Line != b.Line {
		return a.Line - b.Line
	}
	if a.Column != b.Column {
		return a.Column - b.Column
	}
	return 0
}
