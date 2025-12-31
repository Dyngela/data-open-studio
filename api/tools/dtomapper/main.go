package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
)

var (
	typeName = flag.String("type", "", "interface name; must be set")
	output   = flag.String("output", "", "output file name; default {type}.impl.generated.go")
)

func main() {
	flag.Parse()

	if *typeName == "" {
		log.Fatal("missing -type flag")
	}

	// Parse current directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("failed to parse directory: %v", err)
	}

	var interfaceInfo *InterfaceInfo
	var packageName string
	var fileImports []string

	// Find the target interface with generate:dtomapper comment
	for pkgName, pkg := range pkgs {
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}
		packageName = pkgName

		for _, file := range pkg.Files {
			// Extract imports from the source file
			for _, imp := range file.Imports {
				if imp.Path != nil {
					fileImports = append(fileImports, imp.Path.Value)
				}
			}

			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != *typeName {
					return true
				}

				interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					return true
				}

				interfaceInfo = parseInterface(typeSpec.Name.Name, interfaceType, fset)
				return false
			})

			if interfaceInfo != nil {
				break
			}
		}
		if interfaceInfo != nil {
			break
		}
	}

	if interfaceInfo == nil {
		log.Fatalf("interface %s not found", *typeName)
	}

	// Parse all structs in the package for field mapping
	structMap := parseAllStructs(fset, pkgs)

	// Also parse related packages (request, response, models)
	relatedDirs := []string{"../request", "../response", "../models", "../../models"}
	for _, dir := range relatedDirs {
		if relatedPkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments); err == nil {
			for _, structInfo := range parseAllStructs(fset, relatedPkgs) {
				structMap[structInfo.Name] = structInfo
			}
		}
	}

	// Generate mapper implementation
	code := generateMapperImpl(packageName, interfaceInfo, structMap, fileImports)

	// Format the generated code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		log.Fatalf("failed to format generated code: %v\n%s", err, code)
	}

	// Determine output filename
	outputFile := *output
	if outputFile == "" {
		outputFile = strings.ToLower(interfaceInfo.Name) + ".impl.generated.go"
	}

	// Write to file
	if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
		log.Fatalf("failed to write output: %v", err)
	}

	fmt.Printf("Generated %s\n", outputFile)
}

type InterfaceInfo struct {
	Name    string
	Methods []MethodInfo
}

type MethodInfo struct {
	Name       string
	Params     []ParamInfo
	Returns    []string
	IsUpdate   bool
	DocComment string
}

type ParamInfo struct {
	Name string
	Type string
}

type StructInfo struct {
	Name               string
	Package            string
	Fields             []FieldInfo
	FullyQualifiedName string
}

type FieldInfo struct {
	Name     string
	Type     string
	IsPtr    bool
	BaseType string
}

func parseInterface(name string, iface *ast.InterfaceType, fset *token.FileSet) *InterfaceInfo {
	info := &InterfaceInfo{
		Name:    name,
		Methods: []MethodInfo{},
	}

	for _, method := range iface.Methods.List {
		if len(method.Names) == 0 {
			continue // Skip embedded interfaces
		}

		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		methodInfo := MethodInfo{
			Name:    method.Names[0].Name,
			Params:  []ParamInfo{},
			Returns: []string{},
		}

		// Check for --update comment
		if method.Doc != nil {
			for _, comment := range method.Doc.List {
				methodInfo.DocComment = comment.Text
				if strings.Contains(comment.Text, "--update") {
					methodInfo.IsUpdate = true
				}
			}
		}

		// Parse parameters
		if funcType.Params != nil {
			paramIndex := 0
			for _, param := range funcType.Params.List {
				paramType := exprToString(param.Type)
				if len(param.Names) > 0 {
					// Named parameters
					for _, name := range param.Names {
						methodInfo.Params = append(methodInfo.Params, ParamInfo{
							Name: name.Name,
							Type: paramType,
						})
					}
				} else {
					// Unnamed parameter - generate a name
					paramIndex++
					methodInfo.Params = append(methodInfo.Params, ParamInfo{
						Name: fmt.Sprintf("arg%d", paramIndex),
						Type: paramType,
					})
				}
			}
		}

		// Parse return types
		if funcType.Results != nil {
			for _, result := range funcType.Results.List {
				methodInfo.Returns = append(methodInfo.Returns, exprToString(result.Type))
			}
		}

		info.Methods = append(info.Methods, methodInfo)
	}

	return info
}

func parseAllStructs(fset *token.FileSet, pkgs map[string]*ast.Package) map[string]*StructInfo {
	structMap := make(map[string]*StructInfo)

	for pkgName, pkg := range pkgs {
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return true
				}

				structInfo := parseStruct(typeSpec.Name.Name, pkgName, structType)
				structMap[typeSpec.Name.Name] = structInfo
				structMap[pkgName+"."+typeSpec.Name.Name] = structInfo

				return true
			})
		}
	}

	return structMap
}

func parseStruct(name, pkg string, st *ast.StructType) *StructInfo {
	info := &StructInfo{
		Name:               name,
		Package:            pkg,
		Fields:             []FieldInfo{},
		FullyQualifiedName: pkg + "." + name,
	}

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue // Skip embedded fields
		}

		fieldName := field.Names[0].Name
		fieldType := exprToString(field.Type)

		fieldInfo := FieldInfo{
			Name: fieldName,
			Type: fieldType,
		}

		// Check if pointer
		if strings.HasPrefix(fieldType, "*") {
			fieldInfo.IsPtr = true
			fieldInfo.BaseType = strings.TrimPrefix(fieldType, "*")
		} else {
			fieldInfo.BaseType = fieldType
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	return info
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	default:
		return ""
	}
}

const implTemplate = `// Code generated by dtomapper -type={{.InterfaceName}}; DO NOT EDIT.

package {{.Package}}

import (
{{range .Imports}}	{{.}}
{{end}})

// {{.ImplName}} implements {{.InterfaceName}}
type {{.ImplName}} struct{}

// New{{.InterfaceName}} creates a new instance of {{.ImplName}}
func New{{.InterfaceName}}() {{.InterfaceName}} {
	return &{{.ImplName}}{}
}
{{range .Methods}}
// {{.Name}} {{.Comment}}
func (m *{{$.ImplName}}) {{.Name}}({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}) {{if .Returns}}{{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}{{end}} {
{{.Body}}
}
{{end}}
`

func generateMapperImpl(packageName string, iface *InterfaceInfo, structMap map[string]*StructInfo, fileImports []string) string {
	tmpl, err := template.New("impl").Parse(implTemplate)
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	implName := iface.Name + "Impl"

	// Generate method implementations
	var methods []map[string]interface{}
	for _, method := range iface.Methods {
		body := generateMethodBody(method, structMap)

		methods = append(methods, map[string]interface{}{
			"Name":    method.Name,
			"Params":  method.Params,
			"Returns": method.Returns,
			"Body":    body,
			"Comment": strings.TrimPrefix(method.DocComment, "//"),
		})
	}

	// Use imports from the source file
	var importSlice []string
	for _, imp := range fileImports {
		if imp != "" {
			importSlice = append(importSlice, imp)
		}
	}

	data := struct {
		Package       string
		InterfaceName string
		ImplName      string
		Imports       []string
		Methods       []map[string]interface{}
	}{
		Package:       packageName,
		InterfaceName: iface.Name,
		ImplName:      implName,
		Imports:       importSlice,
		Methods:       methods,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	return buf.String()
}

func generateMethodBody(method MethodInfo, structMap map[string]*StructInfo) string {
	if len(method.Returns) == 0 {
		return "\t// TODO: implement\n\tpanic(\"not implemented\")"
	}

	if len(method.Params) == 0 {
		return "\t// TODO: implement (no parameters)\n\tpanic(\"not implemented\")"
	}

	sourceType := method.Params[0].Type
	targetType := method.Returns[0]

	// Clean up types and extract package prefixes
	sourceType = strings.TrimPrefix(sourceType, "*")
	targetType = strings.TrimPrefix(targetType, "*")

	// Extract package prefix from target type (e.g., "models.Vehicle" -> "models")
	targetPackagePrefix := ""
	if idx := strings.LastIndex(targetType, "."); idx >= 0 {
		targetPackagePrefix = targetType[:idx+1]
	}

	sourceStruct := findStruct(sourceType, structMap)
	targetStruct := findStruct(targetType, structMap)

	if sourceStruct == nil || targetStruct == nil {
		return fmt.Sprintf("\t// TODO: Could not find struct definitions for %s or %s\n\tpanic(\"not implemented\")", sourceType, targetType)
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("\tvar result %s\n", targetType))

	// Build field mapping
	targetFieldMap := make(map[string]*FieldInfo)
	for i := range targetStruct.Fields {
		targetFieldMap[targetStruct.Fields[i].Name] = &targetStruct.Fields[i]
	}

	paramName := method.Params[0].Name

	for _, sourceField := range sourceStruct.Fields {
		targetField, exists := targetFieldMap[sourceField.Name]
		if !exists {
			continue
		}

		// Handle slice fields with nested struct mapping
		if strings.HasPrefix(sourceField.Type, "[]") && strings.HasPrefix(targetField.Type, "[]") {
			sourceElemType := strings.TrimPrefix(sourceField.Type, "[]")
			targetElemType := strings.TrimPrefix(targetField.Type, "[]")

			// Apply package prefix if target element type doesn't have one
			targetElemTypeForMake := targetElemType
			if !strings.Contains(targetElemType, ".") && targetPackagePrefix != "" {
				targetElemTypeForMake = targetPackagePrefix + targetElemType
			}

			sourceElemStruct := findStruct(sourceElemType, structMap)
			targetElemStruct := findStruct(targetElemType, structMap)

			if sourceElemStruct != nil && targetElemStruct != nil {
				// Map fields within the slice element
				targetElemFieldMap := make(map[string]*FieldInfo)
				for i := range targetElemStruct.Fields {
					targetElemFieldMap[targetElemStruct.Fields[i].Name] = &targetElemStruct.Fields[i]
				}

				// Collect field mappings
				var fieldMappings []string
				for _, srcField := range sourceElemStruct.Fields {
					tgtField, exists := targetElemFieldMap[srcField.Name]
					if !exists {
						continue
					}

					// Skip nested slices for now
					if strings.HasPrefix(srcField.Type, "[]") || strings.HasPrefix(tgtField.Type, "[]") {
						continue
					}

					// Handle pointer conversions in slice elements
					if tgtField.IsPtr && !srcField.IsPtr {
						fieldMappings = append(fieldMappings,
							fmt.Sprintf("\t\t\tval%s := src.%s\n\t\t\tresult.%s[i].%s = &val%s",
								srcField.Name, srcField.Name, targetField.Name, tgtField.Name, srcField.Name))
					} else if !tgtField.IsPtr && srcField.IsPtr {
						fieldMappings = append(fieldMappings,
							fmt.Sprintf("\t\t\tif src.%s != nil {\n\t\t\t\tresult.%s[i].%s = *src.%s\n\t\t\t}",
								srcField.Name, targetField.Name, tgtField.Name, srcField.Name))
					} else {
						fieldMappings = append(fieldMappings,
							fmt.Sprintf("\t\t\tresult.%s[i].%s = src.%s", targetField.Name, tgtField.Name, srcField.Name))
					}
				}

				// Only generate the loop if there are fields to map
				if len(fieldMappings) > 0 {
					buf.WriteString(fmt.Sprintf("\tif len(%s.%s) > 0 {\n", paramName, sourceField.Name))
					buf.WriteString(fmt.Sprintf("\t\tresult.%s = make([]%s, len(%s.%s))\n",
						targetField.Name, targetElemTypeForMake, paramName, sourceField.Name))
					buf.WriteString(fmt.Sprintf("\t\tfor i, src := range %s.%s {\n", paramName, sourceField.Name))

					for _, mapping := range fieldMappings {
						buf.WriteString(mapping + "\n")
					}

					buf.WriteString("\t\t}\n")
					buf.WriteString("\t}\n")
				} else {
					// No fields to map
					buf.WriteString(fmt.Sprintf("\t// TODO: No matching fields to map for slice %s (%s -> %s)\n",
						sourceField.Name, sourceElemType, targetElemType))
				}
				continue
			}

			// If we can't find struct definitions, add TODO
			buf.WriteString(fmt.Sprintf("\t// TODO: Handle slice field %s manually (struct not found)\n", sourceField.Name))
			continue
		} else if strings.HasPrefix(sourceField.Type, "[]") || strings.HasPrefix(targetField.Type, "[]") {
			// Only one side is a slice or types don't match
			buf.WriteString(fmt.Sprintf("\t// TODO: Handle slice field %s manually (type mismatch)\n", sourceField.Name))
			continue
		}

		// Generate mapping based on whether this is an update method
		if method.IsUpdate && sourceField.IsPtr {
			// For update methods with pointer fields, check for nil
			buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", paramName, sourceField.Name))

			// Handle pointer type mismatches
			if targetField.IsPtr && sourceField.IsPtr {
				// Both pointers: direct assignment
				buf.WriteString(fmt.Sprintf("\t\tresult.%s = %s.%s\n", targetField.Name, paramName, sourceField.Name))
			} else if !targetField.IsPtr && sourceField.IsPtr {
				// Source is pointer, target is not: dereference
				buf.WriteString(fmt.Sprintf("\t\tresult.%s = *%s.%s\n", targetField.Name, paramName, sourceField.Name))
			} else if targetField.IsPtr && !sourceField.IsPtr {
				// Source is not pointer, target is: take address
				buf.WriteString(fmt.Sprintf("\t\tval := %s.%s\n", paramName, sourceField.Name))
				buf.WriteString(fmt.Sprintf("\t\tresult.%s = &val\n", targetField.Name))
			}

			buf.WriteString("\t}\n")
		} else {
			// For create methods or non-pointer fields, direct assignment
			if targetField.IsPtr && !sourceField.IsPtr {
				// Target is pointer, source is not
				buf.WriteString(fmt.Sprintf("\tval%s := %s.%s\n", sourceField.Name, paramName, sourceField.Name))
				buf.WriteString(fmt.Sprintf("\tresult.%s = &val%s\n", targetField.Name, sourceField.Name))
			} else if !targetField.IsPtr && sourceField.IsPtr {
				// Target is not pointer, source is
				buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", paramName, sourceField.Name))
				buf.WriteString(fmt.Sprintf("\t\tresult.%s = *%s.%s\n", targetField.Name, paramName, sourceField.Name))
				buf.WriteString("\t}\n")
			} else {
				// Both same pointer status
				buf.WriteString(fmt.Sprintf("\tresult.%s = %s.%s\n", targetField.Name, paramName, sourceField.Name))
			}
		}
	}

	buf.WriteString("\treturn result\n")
	return buf.String()
}

func findStruct(typeName string, structMap map[string]*StructInfo) *StructInfo {
	// Try direct lookup
	if s, ok := structMap[typeName]; ok {
		return s
	}

	// Try without package prefix
	parts := strings.Split(typeName, ".")
	if len(parts) > 1 {
		if s, ok := structMap[parts[len(parts)-1]]; ok {
			return s
		}
	}

	return nil
}
