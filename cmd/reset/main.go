package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fieldInfo struct {
	Name string
	Expr ast.Expr
}

type structInfo struct {
	Name   string
	Fields []fieldInfo
}

type packageInfo struct {
	Name    string
	Structs []structInfo
	Imports map[string]string
}

type codeLine struct {
	indent int
	text   string
}

type generator struct {
	imports map[string]string
	used    map[string]struct{}
}

func newGenerator(imports map[string]string) *generator {
	return &generator{
		imports: imports,
		used:    make(map[string]struct{}),
	}
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "reset generator: %v\n", err)
		os.Exit(1)
	}

	pkgs := make(map[string]*packageInfo)

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		switch name {
		case ".git", "vendor":
			return filepath.SkipDir
		}
		if name == "testdata" {
			return filepath.SkipDir
		}

		info, err := scanDir(path)
		if err != nil {
			return err
		}
		if info == nil || len(info.Structs) == 0 {
			return nil
		}
		pkgs[path] = info
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "reset generator: %v\n", err)
		os.Exit(1)
	}

	for dir, pkg := range pkgs {
		src, err := buildSource(pkg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reset generator: %v\n", err)
			os.Exit(1)
		}

		formatted, err := format.Source(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reset generator: %v\n", err)
			os.Exit(1)
		}

		target := filepath.Join(dir, "reset.gen.go")

		if existing, err := os.ReadFile(target); err == nil {
			if bytes.Equal(existing, formatted) {
				continue
			}
		}

		if err := os.WriteFile(target, formatted, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "reset generator: %v\n", err)
			os.Exit(1)
		}
	}
}

func scanDir(dir string) (*packageInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var goFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") || name == "reset.gen.go" {
			continue
		}
		goFiles = append(goFiles, filepath.Join(dir, name))
	}

	if len(goFiles) == 0 {
		return nil, nil
	}

	fset := token.NewFileSet()
	files := make(map[string]*ast.File)
	for _, file := range goFiles {
		parsed, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files[file] = parsed
	}

	var pkgName string
	var structs []structInfo
	imports := make(map[string]string)

	for _, file := range files {
		if pkgName == "" {
			pkgName = file.Name.Name
		}
		for _, spec := range file.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if spec.Name != nil {
				alias := spec.Name.Name
				if alias == "_" || alias == "." {
					continue
				}
				if _, exists := imports[alias]; !exists {
					imports[alias] = path
				}
				continue
			}
			parts := strings.Split(path, "/")
			alias := parts[len(parts)-1]
			if alias == "" {
				continue
			}
			if _, exists := imports[alias]; !exists {
				imports[alias] = path
			}
		}

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				if !hasResetComment(typeSpec.Doc, genDecl.Doc) {
					continue
				}

				var fields []fieldInfo

				if structType.Fields != nil {
					for _, field := range structType.Fields.List {
						if len(field.Names) == 0 {
							continue
						}
						for _, name := range field.Names {
							fields = append(fields, fieldInfo{
								Name: name.Name,
								Expr: field.Type,
							})
						}
					}
				}

				structs = append(structs, structInfo{
					Name:   typeSpec.Name.Name,
					Fields: fields,
				})
			}
		}
	}

	if len(structs) == 0 {
		return nil, nil
	}

	return &packageInfo{
		Name:    pkgName,
		Structs: structs,
		Imports: imports,
	}, nil
}

func hasResetComment(groups ...*ast.CommentGroup) bool {
	for _, group := range groups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			if strings.Contains(comment.Text, "generate:reset") {
				return true
			}
		}
	}
	return false
}

func buildSource(pkg *packageInfo) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("package " + pkg.Name + "\n\n")

	gen := newGenerator(pkg.Imports)
	var methods [][]codeLine

	for i, st := range pkg.Structs {
		if i > 0 {
			buf.WriteString("\n")
		}
		lines := generateResetMethod(gen, st)
		methods = append(methods, lines)
	}

	if len(gen.used) > 0 {
		buf.WriteString("import (\n")
		aliases := make([]string, 0, len(gen.used))
		for alias := range gen.used {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		for _, alias := range aliases {
			spec, ok := gen.imports[alias]
			if !ok {
				continue
			}
			if alias == importBase(spec) {
				buf.WriteString(fmt.Sprintf("\t%q\n", spec))
			} else {
				buf.WriteString(fmt.Sprintf("\t%s %q\n", alias, spec))
			}
		}
		buf.WriteString(")\n\n")
	}

	for i, lines := range methods {
		if i > 0 {
			buf.WriteString("\n")
		}
		for _, line := range lines {
			buf.WriteString(strings.Repeat("\t", line.indent))
			buf.WriteString(line.text)
			buf.WriteString("\n")
		}
	}

	return buf.Bytes(), nil
}

func generateResetMethod(gen *generator, st structInfo) []codeLine {
	recv := receiverName(st.Name)
	header := fmt.Sprintf("func (%s *%s) Reset() {", recv, st.Name)
	lines := []codeLine{
		{0, header},
		{1, fmt.Sprintf("if %s == nil {", recv)},
		{2, "return"},
		{1, "}"},
	}

	for _, field := range st.Fields {
		fieldExpr := fmt.Sprintf("%s.%s", recv, field.Name)
		lines = append(lines, generateFieldLines(gen, fieldExpr, field.Expr)...)
	}

	lines = append(lines, codeLine{0, "}"})
	return lines
}

func receiverName(structName string) string {
	if structName == "" {
		return "r"
	}
	r := strings.ToLower(string(structName[0]))
	if r < "a" || r > "z" {
		return "r"
	}
	return r
}

func generateFieldLines(gen *generator, fieldExpr string, expr ast.Expr) []codeLine {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return generatePointerLines(gen, fieldExpr, t.X)
	default:
		return shiftLines(generateValueLines(gen, fieldExpr, fieldExpr, expr), 1)
	}
}

func generatePointerLines(gen *generator, fieldExpr string, elem ast.Expr) []codeLine {
	lines := []codeLine{
		{1, fmt.Sprintf("if %s != nil {", fieldExpr)},
		{2, fmt.Sprintf("if resetter, ok := interface{}(%s).(interface{ Reset() }); ok {", fieldExpr)},
		{3, "resetter.Reset()"},
	}

	body := generateValueLines(gen, "*"+fieldExpr, fmt.Sprintf("(*%s)", fieldExpr), elem)
	if len(body) > 0 {
		lines = append(lines, codeLine{2, "} else {"})
		lines = append(lines, shiftLines(body, 3)...)
		lines = append(lines, codeLine{2, "}"})
	} else {
		lines = append(lines, codeLine{2, "}"})
	}

	lines = append(lines, codeLine{1, "}"})
	return lines
}

func generateValueLines(gen *generator, target, valueExpr string, expr ast.Expr) []codeLine {
	switch t := expr.(type) {
	case *ast.ArrayType:
		if t.Len == nil {
			return []codeLine{{0, fmt.Sprintf("%s = %s[:0]", target, valueExpr)}}
		}
		gen.noteImportsFromExpr(t)
		return []codeLine{{0, fmt.Sprintf("%s = %s{}", target, gen.exprString(t))}}
	case *ast.MapType:
		return []codeLine{{0, fmt.Sprintf("clear(%s)", valueExpr)}}
	case *ast.Ident:
		if zero, ok := zeroForIdent(t.Name); ok {
			return []codeLine{{0, fmt.Sprintf("%s = %s", target, zero)}}
		}
		return []codeLine{{0, fmt.Sprintf("%s = %s{}", target, t.Name)}}
	case *ast.SelectorExpr:
		gen.noteImportsFromExpr(t)
		return []codeLine{{0, fmt.Sprintf("%s = %s{}", target, gen.exprString(t))}}
	case *ast.InterfaceType:
		return []codeLine{{0, fmt.Sprintf("%s = nil", target)}}
	case *ast.StructType:
		return []codeLine{{0, fmt.Sprintf("%s = %s{}", target, gen.exprString(t))}}
	default:
		gen.noteImportsFromExpr(expr)
		typeStr := gen.exprString(expr)
		return []codeLine{
			{0, fmt.Sprintf("var zero %s", typeStr)},
			{0, fmt.Sprintf("%s = zero", target)},
		}
	}
}

func shiftLines(lines []codeLine, delta int) []codeLine {
	result := make([]codeLine, len(lines))
	for i, line := range lines {
		result[i] = codeLine{
			indent: line.indent + delta,
			text:   line.text,
		}
	}
	return result
}

func (g *generator) exprString(expr ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), expr)
	return buf.String()
}

func (g *generator) noteImportsFromExpr(expr ast.Expr) {
	if g == nil {
		return
	}
	ast.Inspect(expr, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok {
			if g.imports == nil {
				return true
			}
			if _, exists := g.imports[ident.Name]; exists {
				g.used[ident.Name] = struct{}{}
			}
		}
		return true
	})
}

func zeroForIdent(name string) (string, bool) {
	switch name {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64",
		"complex64", "complex128":
		return "0", true
	case "string":
		return "\"\"", true
	case "bool":
		return "false", true
	case "byte":
		return "0", true
	case "rune":
		return "0", true
	case "error", "any", "interface{}":
		return "nil", true
	default:
		return "", false
	}
}

func importBase(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
