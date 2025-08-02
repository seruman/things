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

	if flagVimgrep {
		if flagFormat != "" {
			return fmt.Errorf("cannot use -vimgrep and -format together")
		}

		flagFormat = "{{.RelativeFileName}}:{{.Range.Start.Line}}:{{.Range.Start.Column}}:{{.Package}}:{{.FullName}}"
	}

	if flagFormat == "" {
		flagFormat = "{{.FullDisplayName}}"
	}

	tmpl, err := template.New("format").Parse(flagFormat)
	if err != nil {
		return fmt.Errorf("failed to parse format: %w", err)
	}

	for test := range tests {
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
) (iter.Seq[*TestInfo], error) {
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

	return func(yield func(*TestInfo) bool) {
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

			// TODO: To not get panic when running on for file(s) not in a module.
			pkgPath := pkg.PkgPath
			packageName := pkgPath
			if pkg.Module != nil {
				moduleName := pkg.Module.Path
				packageName = strings.TrimPrefix(pkgPath, moduleName+"/")
			}
			directory := pkg.Dir

			inspect := inspector.New(testFiles)

			finder := newTestFinder(pkg.Fset, packageName, directory, logger)
			for test := range finder.find(inspect) {
				if !yield(test) {
					return
				}
			}
		}
	}, nil
}

type tableTestInfo struct {
	name  string
	start token.Position
	end   token.Position
}

type scope map[string]ast.Node

type testFinder struct {
	fset      *token.FileSet
	pkgName   string
	directory string
	logger    func(string, ...any)

	scopeStack []scope
	testStack  []*TestInfo
}

func newTestFinder(fset *token.FileSet, pkgName, dir string, logger func(string, ...any)) *testFinder {
	return &testFinder{
		fset:       fset,
		pkgName:    pkgName,
		directory:  dir,
		logger:     logger,
		scopeStack: []scope{make(scope)},
	}
}

func (tf *testFinder) pushScope() {
	tf.scopeStack = append(tf.scopeStack, make(scope))
}

func (tf *testFinder) pushTest(test *TestInfo) {
	tf.testStack = append(tf.testStack, test)
}

func (tf *testFinder) popTest() {
	if len(tf.testStack) > 0 {
		tf.testStack = tf.testStack[:len(tf.testStack)-1]
	}
}

func (tf *testFinder) currentTest() *TestInfo {
	if len(tf.testStack) > 0 {
		return tf.testStack[len(tf.testStack)-1]
	}
	return nil
}

func (tf *testFinder) popScope() {
	if len(tf.scopeStack) > 1 {
		tf.scopeStack = tf.scopeStack[:len(tf.scopeStack)-1]
	}
}

func (tf *testFinder) find(inspect *inspector.Inspector) iter.Seq[*TestInfo] {
	return func(yield func(*TestInfo) bool) {
		nodeFilter := []ast.Node{
			(*ast.FuncDecl)(nil),
			(*ast.FuncLit)(nil),
			(*ast.CallExpr)(nil),
			(*ast.AssignStmt)(nil),
			(*ast.RangeStmt)(nil),
		}

		shouldStop := false

		// TODO: Tracking these two is quite shit tbh.
		runCallbacks := make(map[*ast.FuncLit]*TestInfo)
		funcLitHasTest := make(map[*ast.FuncLit]struct{})

		inspect.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
			if shouldStop {
				return false
			}

			switch n := node.(type) {
			case *ast.FuncDecl:
				if push {
					if test := tf.handleFuncDecl(n); test != nil {
						if !yield(test) {
							shouldStop = true
							return false
						}

						tf.pushTest(test)
					}
				} else {
					tf.popScope()
					if len(tf.testStack) > 0 {
						tf.popTest()
					}
				}

			case *ast.FuncLit:
				if push {
					if test, ok := runCallbacks[n]; ok {
						tf.pushScope()
						tf.pushTest(test)
						funcLitHasTest[n] = struct{}{}
						delete(runCallbacks, n)
					} else {
						tf.pushScope()
					}
				} else {
					tf.popScope()
					if _, ok := funcLitHasTest[n]; ok {
						tf.popTest()
						delete(funcLitHasTest, n)
					}
				}

			case *ast.CallExpr:
				if push {
					// TODO: returned subtest just used as a signal, just a
					// bool should be OK.
					subTests, funcLitTest := tf.handleCallExpr(n, yield)
					if !subTests {
						shouldStop = true
						return false
					}

					if funcLitTest != nil {
						if funcLit, ok := n.Args[1].(*ast.FuncLit); ok {
							runCallbacks[funcLit] = funcLitTest
						}
					}
				}

			case *ast.AssignStmt:
				if push {
					tf.handleAssignStmt(n)
				}
			case *ast.RangeStmt:
				if push {
					tf.handleRangeStmt(n)
				}
			}

			return true
		})
	}
}

func (tf *testFinder) handleAssignStmt(n *ast.AssignStmt) {
	if len(n.Lhs) == 1 && len(n.Rhs) == 1 {
		if ident, ok := n.Lhs[0].(*ast.Ident); ok {
			if len(tf.scopeStack) > 0 {
				tf.scopeStack[len(tf.scopeStack)-1][ident.Name] = n.Rhs[0]
			}
		}
	}
}

func (tf *testFinder) handleRangeStmt(n *ast.RangeStmt) {
	if len(tf.scopeStack) == 0 {
		return
	}
	currentScope := tf.scopeStack[len(tf.scopeStack)-1]

	if n.Key != nil {
		if ident, ok := n.Key.(*ast.Ident); ok {
			currentScope[ident.Name] = n.X
		}
	}

	if n.Value != nil {
		if ident, ok := n.Value.(*ast.Ident); ok {
			currentScope[ident.Name] = n.X
		}
	}
}

func (tf *testFinder) handleFuncDecl(n *ast.FuncDecl) *TestInfo {
	// New function, push a new scope to isolate assignments.
	tf.pushScope()

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
	}

	return test
}

func (tf *testFinder) handleCallExpr(n *ast.CallExpr, yield func(*TestInfo) bool) (bool, *TestInfo) {
	if !tf.isRunCall(n) {
		return true, nil
	}

	parent := tf.currentTest()
	if parent == nil {
		return true, nil
	}

	subTests := tf.createSubTests(n, parent)
	if len(subTests) == 0 {
		return true, nil
	}

	for _, subTest := range subTests {
		if !yield(subTest) {
			return false, nil
		}
	}

	// TODO:
	if funcLit, ok := n.Args[1].(*ast.FuncLit); ok && funcLit.Body != nil && len(subTests) > 0 {
		return true, subTests[len(subTests)-1]
	}

	return true, nil
}

func (tf *testFinder) isRunCall(n *ast.CallExpr) bool {
	selExpr, ok := n.Fun.(*ast.SelectorExpr)
	return ok && selExpr.Sel.Name == "Run" && len(n.Args) >= 2
}

func (tf *testFinder) createSubTests(n *ast.CallExpr, parent *TestInfo) []*TestInfo {
	filename := tf.fset.Position(n.Pos()).Filename
	start := tf.fset.Position(n.Pos())
	end := tf.fset.Position(n.End())

	switch arg := n.Args[0].(type) {
	case *ast.BasicLit:
		if arg.Kind == token.STRING {
			subtestName := strings.Trim(arg.Value, "\"'`")
			subTest := tf.createNamedSubTest(subtestName, parent, filename, start, end)
			return []*TestInfo{subTest}
		}
	case *ast.SelectorExpr:
		if tableTests := extractTableTestNames(arg, tf.scopeStack, tf.fset); len(tableTests) > 0 {
			var subTests []*TestInfo
			for _, tt := range tableTests {
				subTest := tf.createNamedSubTest(tt.name, parent, filename, tt.start, tt.end)
				subTests = append(subTests, subTest)
			}
			return subTests
		}
		subTest := tf.createGeneratedSubTest(arg, parent, filename, start, end)
		return []*TestInfo{subTest}
	default:
		subTest := tf.createGeneratedSubTest(arg, parent, filename, start, end)
		return []*TestInfo{subTest}
	}
	return nil
}

func extractTableTestNames(selector *ast.SelectorExpr, scopeStack []scope, fset *token.FileSet) []tableTestInfo {
	// Get the variable name e.g. "c" from `c.name`.
	varIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return nil
	}

	fieldName := selector.Sel.Name

	rangeExpr := lookupInScope(varIdent.Name, scopeStack)
	if rangeExpr == nil {
		return nil
	}

	sliceExpr := rangeExpr
	if ident, ok := rangeExpr.(*ast.Ident); ok {
		sliceExpr = lookupInScope(ident.Name, scopeStack)
		if sliceExpr == nil {
			return nil
		}
	}

	compLit, ok := sliceExpr.(*ast.CompositeLit)
	if !ok {
		return nil
	}

	// Try to extract field positions from the type 🤞.
	// TODO: introducing types would make this more easier, but it would be
	// slower.
	var fieldPositions map[string]int
	if compLit.Type != nil {
		fieldPositions = extractStructFieldPositions(compLit.Type)
	}

	var tableTests []tableTestInfo
	for _, elt := range compLit.Elts {
		if structLit, ok := elt.(*ast.CompositeLit); ok {
			if name := extractFieldValue(structLit, fieldName, fieldPositions); name != "" {
				start := fset.Position(structLit.Pos())
				end := fset.Position(structLit.End())
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

func lookupInScope(name string, scopeStack []scope) ast.Node {
	for i := len(scopeStack) - 1; i >= 0; i-- {
		if node, ok := scopeStack[i][name]; ok {
			return node
		}
	}
	return nil
}

func extractStructFieldPositions(typeExpr ast.Expr) map[string]int {
	positions := make(map[string]int)

	// []struct{...}
	arr, ok := typeExpr.(*ast.ArrayType)
	if !ok {
		return positions
	}

	structType, ok := arr.Elt.(*ast.StructType)
	if !ok {
		return positions
	}

	pos := 0
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			// Anonymous field.
			pos++
			continue
		}

		// Named fields.
		for _, name := range field.Names {
			positions[name.Name] = pos
			pos++
		}
	}

	return positions
}

func extractFieldValue(structLit *ast.CompositeLit, fieldName string, fieldPositions map[string]int) string {
	// Try to find named field (e.g., {name: "test"})
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

	if len(structLit.Elts) == 0 {
		return ""
	}

	// Try positional initialization.
	allPositional := true
	for _, elt := range structLit.Elts {
		if _, ok := elt.(*ast.KeyValueExpr); ok {
			allPositional = false
			break
		}
	}

	if !allPositional {
		return ""
	}

	if fieldPositions != nil {
		if pos, ok := fieldPositions[fieldName]; ok && pos < len(structLit.Elts) {
			if lit, ok := structLit.Elts[pos].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				return strings.Trim(lit.Value, "\"'`")
			}
		}

		return ""
	}

	// If the first field is a string literal, it's likely the
	// field we're looking for; test names are hopefully always
	// strings.
	if lit, ok := structLit.Elts[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return strings.Trim(lit.Value, "\"'`")
	}

	return ""
}

func (tf *testFinder) createNamedSubTest(name string, parent *TestInfo, filename string, start, end token.Position) *TestInfo {
	sanitizedName := rewriteSubTestName(name)
	fullName := parent.FullName + "/" + name
	sanitizedFullName := parent.FullDisplayName + "/" + sanitizedName

	return &TestInfo{
		Name:            name,
		DisplayName:     sanitizedName,
		FullName:        fullName,
		FullDisplayName: sanitizedFullName,
		Package:         parent.Package,
		Directory:       parent.Directory,
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

func (tf *testFinder) createGeneratedSubTest(arg ast.Expr, parent *TestInfo, filename string, start, end token.Position) *TestInfo {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, tf.fset, arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error printing argument: %v\n", err)
		return nil
	}
	subtestName := fmt.Sprintf("<%s>", strings.TrimSpace(buf.String()))
	fullName := fmt.Sprintf("%s/%s", parent.FullName, subtestName)
	fullDisplayName := fmt.Sprintf("%s/%s", parent.FullDisplayName, subtestName)

	return &TestInfo{
		Name:            subtestName,
		DisplayName:     subtestName,
		FullName:        fullName,
		FullDisplayName: fullDisplayName,
		Package:         parent.Package,
		Directory:       parent.Directory,
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
