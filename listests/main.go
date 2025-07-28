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

	"golang.org/x/tools/go/ast/inspector"
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
		flagDir     string
	)

	fs.StringVar(&flagTags, "tags", "", "comma-separated list of build tags to apply")
	fs.BoolVar(&flagVerbose, "v", false, "verbose mode")
	fs.BoolVar(&flagVimgrep, "vimgrep", false, "output in ripgrep's vimgrep format")
	fs.StringVar(&flagFormat, "format", "", "output format")
	fs.StringVar(&flagDir, "dir", ".", "directory to run in")

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
		flagDir,
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

		flagFormat = "{{.RelativeFileName}}:{{.Range.Start.Line}}:{{.Range.Start.Column}}:{{.Package}}:{{.FullName}}"
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

		var testFiles []*ast.File
		for _, file := range pkg.Syntax {
			filename := pkg.Fset.Position(file.Pos()).Filename
			if strings.HasSuffix(filename, "_test.go") {
				testFiles = append(testFiles, file)
			}
		}

		if len(testFiles) == 0 {
			continue
		}

		moduleName := pkg.Module.Path
		pkgPath := pkg.PkgPath
		packageName := strings.TrimPrefix(pkgPath, moduleName+"/")
		directory := pkg.Dir

		inspect := inspector.New(testFiles)

		finder := newTestFinder(pkg.Fset, packageName, directory, logger)
		allTests = append(allTests, finder.find(inspect)...)
	}

	return allTests, nil
}

type tableTestInfo struct {
	name  string
	start token.Position
	end   token.Position
}

type testFinder struct {
	fset      *token.FileSet
	pkgName   string
	directory string
	logger    func(string, ...any)

	testMap     map[ast.Node]*TestInfo
	assignments map[string]ast.Node
}

func newTestFinder(fset *token.FileSet, pkgName, dir string, logger func(string, ...any)) *testFinder {
	return &testFinder{
		fset:        fset,
		pkgName:     pkgName,
		directory:   dir,
		logger:      logger,
		testMap:     make(map[ast.Node]*TestInfo),
		assignments: make(map[string]ast.Node),
	}
}

func (tf *testFinder) find(inspect *inspector.Inspector) []*TestInfo {
	var allTests []*TestInfo

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.RangeStmt)(nil),
	}

	inspect.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		switch n := node.(type) {
		case *ast.FuncDecl:
			if test := tf.handleFuncDecl(n); test != nil {
				allTests = append(allTests, test)
			}
		case *ast.CallExpr:
			tf.handleCallExpr(n, stack)
		case *ast.AssignStmt:
			tf.handleAssignStmt(n)
		case *ast.RangeStmt:
			tf.handleRangeStmt(n)
		}

		return true
	})

	return allTests
}

func (tf *testFinder) handleAssignStmt(n *ast.AssignStmt) {
	// cases := []struct{...}
	if len(n.Lhs) == 1 && len(n.Rhs) == 1 {
		if ident, ok := n.Lhs[0].(*ast.Ident); ok {
			tf.assignments[ident.Name] = n.Rhs[0]
		}
	}
}

func (tf *testFinder) handleRangeStmt(n *ast.RangeStmt) {
	if n.Key != nil {
		if ident, ok := n.Key.(*ast.Ident); ok {
			tf.assignments[ident.Name] = n.X
		}
	}

	if n.Value != nil {
		if ident, ok := n.Value.(*ast.Ident); ok {
			tf.assignments[ident.Name] = n.X
		}
	}
}

func (tf *testFinder) handleFuncDecl(n *ast.FuncDecl) *TestInfo {
	// New function, clear assignments.
	// TODO: If subtest is using a variable from an outer scope, information is
	// lost.
	tf.assignments = make(map[string]ast.Node)

	if n.Name == nil || !strings.HasPrefix(n.Name.Name, "Test") || !isTestFunction(n) {
		return nil
	}

	filename := tf.fset.Position(n.Pos()).Filename
	tf.logger("Processing %s in package %s...\n", filename, tf.pkgName)

	start := tf.fset.Position(n.Name.Pos())
	end := tf.fset.Position(n.End())

	test := &TestInfo{
		Name:            n.Name.Name,
		DisplayName:     n.Name.Name,
		FullName:        n.Name.Name,
		FullDisplayName: n.Name.Name,
		Package:         tf.pkgName,
		Directory:       tf.directory,
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
		IsSubtest:        false,
		SubTests:         nil,
	}

	tf.testMap[n] = test
	return test
}

func (tf *testFinder) handleCallExpr(n *ast.CallExpr, stack []ast.Node) {
	if !tf.isRunCall(n) {
		return
	}

	parentTest := tf.findParentTest(stack)
	if parentTest == nil {
		return
	}

	subTests := tf.createSubTests(n, parentTest)
	if len(subTests) == 0 {
		return
	}

	parentTest.SubTests = append(parentTest.SubTests, subTests...)

	// Map the function literal body to the last subtest so nested t.Run calls can
	// find it. For table tests, all subtests share the same t.Run call.
	if funcLit, ok := n.Args[1].(*ast.FuncLit); ok && funcLit.Body != nil {
		tf.testMap[funcLit.Body] = subTests[len(subTests)-1]
	}
}

func (tf *testFinder) isRunCall(n *ast.CallExpr) bool {
	selExpr, ok := n.Fun.(*ast.SelectorExpr)
	return ok && selExpr.Sel.Name == "Run" && len(n.Args) >= 2
}

func (tf *testFinder) findParentTest(stack []ast.Node) *TestInfo {
	for i := len(stack) - 1; i >= 0; i-- {
		if test, exists := tf.testMap[stack[i]]; exists {
			return test
		}
	}
	return nil
}

func (tf *testFinder) createSubTests(n *ast.CallExpr, parentTest *TestInfo) []*TestInfo {
	filename := tf.fset.Position(n.Pos()).Filename
	start := tf.fset.Position(n.Pos())
	end := tf.fset.Position(n.End())

	switch arg := n.Args[0].(type) {
	case *ast.BasicLit:
		if arg.Kind == token.STRING {
			subtestName := strings.Trim(arg.Value, "\"'`")
			subTest := tf.createNamedSubTest(subtestName, parentTest, filename, start, end)
			return []*TestInfo{subTest}
		}
	case *ast.SelectorExpr:
		if tableTests := tf.extractTableTestNames(arg); len(tableTests) > 0 {
			var subTests []*TestInfo
			for _, tt := range tableTests {
				subTest := tf.createNamedSubTest(tt.name, parentTest, filename, tt.start, tt.end)
				subTests = append(subTests, subTest)
			}
			return subTests
		}
		subTest := tf.createGeneratedSubTest(arg, parentTest, filename, start, end)
		return []*TestInfo{subTest}
	default:
		subTest := tf.createGeneratedSubTest(arg, parentTest, filename, start, end)
		return []*TestInfo{subTest}
	}
	return nil
}

func (tf *testFinder) extractTableTestNames(selector *ast.SelectorExpr) []tableTestInfo {
	// Get the variable name e.g. "c" from `c.name`.
	varIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return nil
	}

	fieldName := selector.Sel.Name

	rangeExpr, ok := tf.assignments[varIdent.Name]
	if !ok {
		return nil
	}

	sliceExpr := rangeExpr
	if ident, ok := rangeExpr.(*ast.Ident); ok {
		// Ranging over a variable, e.g. `for _, c := range cases`.
		sliceExpr, ok = tf.assignments[ident.Name]
		if !ok {
			return nil
		}
	}

	compLit, ok := sliceExpr.(*ast.CompositeLit)
	if !ok {
		return nil
	}

	var tableTests []tableTestInfo
	for _, elt := range compLit.Elts {
		if structLit, ok := elt.(*ast.CompositeLit); ok {
			if name := tf.extractFieldValue(structLit, fieldName); name != "" {
				start := tf.fset.Position(structLit.Pos())
				end := tf.fset.Position(structLit.End())
				tableTests = append(tableTests, tableTestInfo{
					name:  name,
					start: start,
					end:   end,
				})
			}
		}
	}

	return tableTests
}

func (tf *testFinder) extractFieldValue(structLit *ast.CompositeLit, fieldName string) string {
	for _, elt := range structLit.Elts {
		kvExpr, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key, ok := kvExpr.Key.(*ast.Ident)
		if !ok || key.Name != fieldName {
			continue
		}

		if lit, ok := kvExpr.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			return strings.Trim(lit.Value, "\"'`")
		}
	}
	return ""
}

func (tf *testFinder) createNamedSubTest(name string, parentTest *TestInfo, filename string, start, end token.Position) *TestInfo {
	sanitizedName := rewriteSubTestName(name)
	fullName := parentTest.FullName + "/" + name
	sanitizedFullName := parentTest.FullName + "/" + sanitizedName

	return &TestInfo{
		Name:            name,
		DisplayName:     sanitizedName,
		FullName:        fullName,
		FullDisplayName: sanitizedFullName,
		Package:         tf.pkgName,
		Directory:       tf.directory,
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

func (tf *testFinder) createGeneratedSubTest(arg ast.Expr, parentTest *TestInfo, filename string, start, end token.Position) *TestInfo {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, tf.fset, arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error printing argument: %v\n", err)
		return nil
	}
	subtestName := fmt.Sprintf("<%s>", strings.TrimSpace(buf.String()))
	fullName := fmt.Sprintf("%s/%s", parentTest.FullName, subtestName)
	fullDisplayName := fmt.Sprintf("%s/%s", parentTest.FullDisplayName, subtestName)

	return &TestInfo{
		Name:             subtestName,
		DisplayName:      subtestName,
		FullName:         fullName,
		FullDisplayName:  fullDisplayName,
		Package:          tf.pkgName,
		Directory:        tf.directory,
		File:             filename,
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
