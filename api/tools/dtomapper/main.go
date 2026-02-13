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
				// Extract imports only from the file containing the target interface
				for _, imp := range file.Imports {
					if imp.Path != nil {
						fileImports = append(fileImports, imp.Path.Value)
					}
				}
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
	IsPatch    bool
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

		// Check for special comments: update, patch
		if method.Doc != nil {
			for _, comment := range method.Doc.List {
				methodInfo.DocComment = comment.Text
				if strings.Contains(comment.Text, "update") {
					methodInfo.IsUpdate = true
				}
				if strings.Contains(comment.Text, "patch") {
					methodInfo.IsPatch = true
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
		fieldType := qualifyType(exprToString(field.Type), pkg)

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
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
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
// {{.name}} {{.Comment}}
func (mapper *{{$.ImplName}}) {{.name}}({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p.name}} {{$p.Type}}{{end}}) {{if .Returns}}{{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r}}{{end}}{{end}} {
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
			"name":    method.Name,
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
	if len(method.Params) == 0 {
		return "\t// TODO: implement (no parameters)\n\tpanic(\"not implemented\")"
	}

	// Handle update pattern: UpdateDbMetadata(req Request, m *Model)
	// Modifies the target struct pointer with non-nil fields from source
	if method.IsUpdate && len(method.Params) >= 2 && len(method.Returns) == 0 {
		return generateUpdateMethodBody(method, structMap)
	}

	// Handle patch pattern: PatchDbMetadata(req Request) map[string]any
	// Returns a map with non-nil fields from source
	if method.IsPatch && len(method.Params) >= 1 && len(method.Returns) == 1 && strings.HasPrefix(method.Returns[0], "map[") {
		return generatePatchMethodBody(method, structMap)
	}

	if len(method.Returns) == 0 {
		return "\t// TODO: implement\n\tpanic(\"not implemented\")"
	}

	sourceType := method.Params[0].Type
	targetType := method.Returns[0]

	// Handle slice-to-slice mapping: ToResponses([]Entity) []Message
	// Delegates to single-item mapper method
	if strings.HasPrefix(sourceType, "[]") && strings.HasPrefix(targetType, "[]") {
		return generateSliceMapperBody(method, structMap)
	}

	// Clean up types
	sourceType = strings.TrimPrefix(sourceType, "*")
	targetType = strings.TrimPrefix(targetType, "*")

	sourceStruct := findStruct(sourceType, structMap)
	targetStruct := findStruct(targetType, structMap)

	if sourceStruct == nil || targetStruct == nil {
		return fmt.Sprintf("\t// TODO: Could not find struct definitions for %s or %s\n\tpanic(\"not implemented\")", sourceType, targetType)
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("\tvar result %s\n", targetType))

	// build field mapping
	targetFieldMap := make(map[string]*FieldInfo)
	for i := range targetStruct.Fields {
		targetFieldMap[targetStruct.Fields[i].Name] = &targetStruct.Fields[i]
	}

	paramName := method.Params[0].Name

	for i := range sourceStruct.Fields {
		sourceField := &sourceStruct.Fields[i]
		targetField, exists := targetFieldMap[sourceField.Name]
		if !exists {
			continue
		}

		// Handle slice fields
		if strings.HasPrefix(sourceField.Type, "[]") || strings.HasPrefix(targetField.Type, "[]") {
			sliceCode := generateSliceFieldMapping(sourceField, targetField, paramName, "result", structMap, false)
			buf.WriteString(sliceCode)
			continue
		}

		// Handle pointer-to-slice fields (e.g., *[]Type)
		if strings.HasPrefix(sourceField.BaseType, "[]") || strings.HasPrefix(targetField.BaseType, "[]") {
			sliceCode := generatePtrSliceFieldMapping(sourceField, targetField, paramName, "result", structMap)
			buf.WriteString(sliceCode)
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
				// Source is pointer, target is not: dereference (with cast if needed)
				deref := fmt.Sprintf("*%s.%s", paramName, sourceField.Name)
				buf.WriteString(fmt.Sprintf("\t\tresult.%s = %s\n", targetField.Name, castExpr(deref, sourceField.BaseType, targetField.BaseType)))
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
				valExpr := castExpr(fmt.Sprintf("%s.%s", paramName, sourceField.Name), sourceField.BaseType, targetField.BaseType)
				buf.WriteString(fmt.Sprintf("\tval%s := %s\n", sourceField.Name, valExpr))
				buf.WriteString(fmt.Sprintf("\tresult.%s = &val%s\n", targetField.Name, sourceField.Name))
			} else if !targetField.IsPtr && sourceField.IsPtr {
				// Target is not pointer, source is
				buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", paramName, sourceField.Name))
				deref := fmt.Sprintf("*%s.%s", paramName, sourceField.Name)
				buf.WriteString(fmt.Sprintf("\t\tresult.%s = %s\n", targetField.Name, castExpr(deref, sourceField.BaseType, targetField.BaseType)))
				buf.WriteString("\t}\n")
			} else {
				// Both same pointer status
				valExpr := castExpr(fmt.Sprintf("%s.%s", paramName, sourceField.Name), sourceField.BaseType, targetField.BaseType)
				buf.WriteString(fmt.Sprintf("\tresult.%s = %s\n", targetField.Name, valExpr))
			}
		}
	}

	buf.WriteString("\treturn result\n")
	return buf.String()
}

// generateSliceMapperBody generates code for slice-to-slice mapping
// Pattern: ToResponses(entities []models.Entity) []response.Message
// It looks for a corresponding single-item mapper method to delegate to
func generateSliceMapperBody(method MethodInfo, structMap map[string]*StructInfo) string {
	sourceType := strings.TrimPrefix(method.Params[0].Type, "[]")
	targetType := strings.TrimPrefix(method.Returns[0], "[]")
	paramName := method.Params[0].Name

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("\tresult := make(%s, len(%s))\n", method.Returns[0], paramName))
	buf.WriteString(fmt.Sprintf("\tfor i, item := range %s {\n", paramName))

	// Try to find a matching single-item mapper method
	// Look for a method that takes sourceType and returns targetType
	singleMapperName := findSingleMapperMethod(method.Name, sourceType, targetType)
	if singleMapperName != "" {
		buf.WriteString(fmt.Sprintf("\t\tresult[i] = mapper.%s(item)\n", singleMapperName))
	} else {
		// Inline the mapping
		sourceStruct := findStruct(sourceType, structMap)
		targetStruct := findStruct(targetType, structMap)

		if sourceStruct != nil && targetStruct != nil {
			targetFieldMap := make(map[string]*FieldInfo)
			for i := range targetStruct.Fields {
				targetFieldMap[targetStruct.Fields[i].Name] = &targetStruct.Fields[i]
			}

			for _, sourceField := range sourceStruct.Fields {
				targetField, exists := targetFieldMap[sourceField.Name]
				if !exists || strings.HasPrefix(sourceField.Type, "[]") {
					continue
				}

				if targetField.IsPtr && !sourceField.IsPtr {
					buf.WriteString(fmt.Sprintf("\t\tval%s := item.%s\n", sourceField.Name, sourceField.Name))
					buf.WriteString(fmt.Sprintf("\t\tresult[i].%s = &val%s\n", targetField.Name, sourceField.Name))
				} else if !targetField.IsPtr && sourceField.IsPtr {
					buf.WriteString(fmt.Sprintf("\t\tif item.%s != nil {\n", sourceField.Name))
					buf.WriteString(fmt.Sprintf("\t\t\tresult[i].%s = *item.%s\n", targetField.Name, sourceField.Name))
					buf.WriteString("\t\t}\n")
				} else {
					buf.WriteString(fmt.Sprintf("\t\tresult[i].%s = item.%s\n", targetField.Name, sourceField.Name))
				}
			}
		} else {
			buf.WriteString("\t\t// TODO: implement field mapping\n")
			buf.WriteString(fmt.Sprintf("\t\t_ = item // placeholder\n"))
		}
	}

	buf.WriteString("\t}\n")
	buf.WriteString("\treturn result\n")
	return buf.String()
}

// findSingleMapperMethod tries to find a corresponding single-item method name
// e.g., ToMetadataResponses -> ToMetadataResponse
func findSingleMapperMethod(sliceMethodName, sourceType, targetType string) string {
	// Try common patterns: ToXxxs -> ToXxx, ToXxxList -> ToXxx
	if strings.HasSuffix(sliceMethodName, "s") && !strings.HasSuffix(sliceMethodName, "ss") {
		return strings.TrimSuffix(sliceMethodName, "s")
	}
	if strings.HasSuffix(sliceMethodName, "List") {
		return strings.TrimSuffix(sliceMethodName, "List")
	}
	if strings.HasSuffix(sliceMethodName, "es") {
		return strings.TrimSuffix(sliceMethodName, "es")
	}
	return ""
}

// generateUpdateMethodBody generates code for update methods that modify an existing struct
// Pattern: UpdateDbMetadata(req request.UpdateMetadata, m *models.MetadataDatabase)
func generateUpdateMethodBody(method MethodInfo, structMap map[string]*StructInfo) string {
	sourceType := strings.TrimPrefix(method.Params[0].Type, "*")
	targetType := strings.TrimPrefix(method.Params[1].Type, "*")

	sourceStruct := findStruct(sourceType, structMap)
	targetStruct := findStruct(targetType, structMap)

	if sourceStruct == nil || targetStruct == nil {
		return fmt.Sprintf("\t// TODO: Could not find struct definitions for %s or %s\n\tpanic(\"not implemented\")", sourceType, targetType)
	}

	var buf bytes.Buffer
	reqParam := method.Params[0].Name
	targetParam := method.Params[1].Name

	// build target field map for matching
	targetFieldMap := make(map[string]*FieldInfo)
	for i := range targetStruct.Fields {
		targetFieldMap[targetStruct.Fields[i].Name] = &targetStruct.Fields[i]
	}

	for i := range sourceStruct.Fields {
		sourceField := &sourceStruct.Fields[i]
		targetField, exists := targetFieldMap[sourceField.Name]
		if !exists {
			continue
		}

		// Handle slice fields
		if strings.HasPrefix(sourceField.Type, "[]") || strings.HasPrefix(targetField.Type, "[]") {
			sliceCode := generateSliceFieldMapping(sourceField, targetField, reqParam, targetParam, structMap, true)
			buf.WriteString(sliceCode)
			continue
		}

		// Handle pointer-to-slice fields (e.g., *[]Type)
		if strings.HasPrefix(sourceField.BaseType, "[]") || strings.HasPrefix(targetField.BaseType, "[]") {
			sliceCode := generatePtrSliceFieldMapping(sourceField, targetField, reqParam, targetParam, structMap)
			buf.WriteString(sliceCode)
			continue
		}

		// For update methods, only update if source field is non-nil (pointer fields)
		if sourceField.IsPtr {
			buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", reqParam, sourceField.Name))

			if targetField.IsPtr && sourceField.IsPtr {
				// Both pointers: direct assignment
				buf.WriteString(fmt.Sprintf("\t\t%s.%s = %s.%s\n", targetParam, targetField.Name, reqParam, sourceField.Name))
			} else if !targetField.IsPtr && sourceField.IsPtr {
				// Source is pointer, target is not: dereference (with cast if needed)
				deref := fmt.Sprintf("*%s.%s", reqParam, sourceField.Name)
				buf.WriteString(fmt.Sprintf("\t\t%s.%s = %s\n", targetParam, targetField.Name, castExpr(deref, sourceField.BaseType, targetField.BaseType)))
			} else if targetField.IsPtr && !sourceField.IsPtr {
				// Source is not pointer, target is: take address
				buf.WriteString(fmt.Sprintf("\t\tval := %s.%s\n", reqParam, sourceField.Name))
				buf.WriteString(fmt.Sprintf("\t\t%s.%s = &val\n", targetParam, targetField.Name))
			}

			buf.WriteString("\t}\n")
		} else {
			// Non-pointer source field: always assign
			if targetField.IsPtr && !sourceField.IsPtr {
				valExpr := castExpr(fmt.Sprintf("%s.%s", reqParam, sourceField.Name), sourceField.BaseType, targetField.BaseType)
				buf.WriteString(fmt.Sprintf("\tval%s := %s\n", sourceField.Name, valExpr))
				buf.WriteString(fmt.Sprintf("\t%s.%s = &val%s\n", targetParam, targetField.Name, sourceField.Name))
			} else if !targetField.IsPtr && sourceField.IsPtr {
				buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", reqParam, sourceField.Name))
				deref := fmt.Sprintf("*%s.%s", reqParam, sourceField.Name)
				buf.WriteString(fmt.Sprintf("\t\t%s.%s = %s\n", targetParam, targetField.Name, castExpr(deref, sourceField.BaseType, targetField.BaseType)))
				buf.WriteString("\t}\n")
			} else {
				valExpr := castExpr(fmt.Sprintf("%s.%s", reqParam, sourceField.Name), sourceField.BaseType, targetField.BaseType)
				buf.WriteString(fmt.Sprintf("\t%s.%s = %s\n", targetParam, targetField.Name, valExpr))
			}
		}
	}

	return buf.String()
}

// generateSliceFieldMapping generates code for mapping slice fields
// isUpdate: if true, generates code for update pattern (modifying existing struct)
func generateSliceFieldMapping(sourceField, targetField *FieldInfo, srcParam, tgtParam string, structMap map[string]*StructInfo, isUpdate bool) string {
	var buf bytes.Buffer

	sourceElemType := strings.TrimPrefix(sourceField.Type, "[]")
	targetElemType := strings.TrimPrefix(targetField.Type, "[]")

	// Check if element types are the same (simple slice copy)
	if sourceElemType == targetElemType {
		if isUpdate {
			buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", srcParam, sourceField.Name))
			buf.WriteString(fmt.Sprintf("\t\t%s.%s = make(%s, len(%s.%s))\n", tgtParam, targetField.Name, targetField.Type, srcParam, sourceField.Name))
			buf.WriteString(fmt.Sprintf("\t\tcopy(%s.%s, %s.%s)\n", tgtParam, targetField.Name, srcParam, sourceField.Name))
			buf.WriteString("\t}\n")
		} else {
			buf.WriteString(fmt.Sprintf("\tif len(%s.%s) > 0 {\n", srcParam, sourceField.Name))
			buf.WriteString(fmt.Sprintf("\t\t%s.%s = make(%s, len(%s.%s))\n", tgtParam, targetField.Name, targetField.Type, srcParam, sourceField.Name))
			buf.WriteString(fmt.Sprintf("\t\tcopy(%s.%s, %s.%s)\n", tgtParam, targetField.Name, srcParam, sourceField.Name))
			buf.WriteString("\t}\n")
		}
		return buf.String()
	}

	// Element types differ - need to map each element
	sourceElemStruct := findStruct(sourceElemType, structMap)
	targetElemStruct := findStruct(targetElemType, structMap)

	if sourceElemStruct == nil || targetElemStruct == nil {
		buf.WriteString(fmt.Sprintf("\t// TODO: Handle slice field %s manually (element struct not found: %s -> %s)\n",
			sourceField.Name, sourceElemType, targetElemType))
		return buf.String()
	}

	// build target element field map
	targetElemFieldMap := make(map[string]*FieldInfo)
	for i := range targetElemStruct.Fields {
		targetElemFieldMap[targetElemStruct.Fields[i].Name] = &targetElemStruct.Fields[i]
	}

	// Collect field mappings for elements
	var fieldMappings []string
	for _, srcField := range sourceElemStruct.Fields {
		tgtField, exists := targetElemFieldMap[srcField.Name]
		if !exists {
			continue
		}

		// Skip nested slices
		if strings.HasPrefix(srcField.Type, "[]") || strings.HasPrefix(tgtField.Type, "[]") {
			continue
		}

		if tgtField.IsPtr && !srcField.IsPtr {
			fieldMappings = append(fieldMappings,
				fmt.Sprintf("\t\t\tval%s := src.%s\n\t\t\t%s.%s[i].%s = &val%s",
					srcField.Name, srcField.Name, tgtParam, targetField.Name, tgtField.Name, srcField.Name))
		} else if !tgtField.IsPtr && srcField.IsPtr {
			fieldMappings = append(fieldMappings,
				fmt.Sprintf("\t\t\tif src.%s != nil {\n\t\t\t\t%s.%s[i].%s = *src.%s\n\t\t\t}",
					srcField.Name, tgtParam, targetField.Name, tgtField.Name, srcField.Name))
		} else {
			fieldMappings = append(fieldMappings,
				fmt.Sprintf("\t\t\t%s.%s[i].%s = src.%s", tgtParam, targetField.Name, tgtField.Name, srcField.Name))
		}
	}

	if len(fieldMappings) == 0 {
		buf.WriteString(fmt.Sprintf("\t// TODO: No matching fields to map for slice %s (%s -> %s)\n",
			sourceField.Name, sourceElemType, targetElemType))
		return buf.String()
	}

	if isUpdate {
		buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", srcParam, sourceField.Name))
	} else {
		buf.WriteString(fmt.Sprintf("\tif len(%s.%s) > 0 {\n", srcParam, sourceField.Name))
	}
	buf.WriteString(fmt.Sprintf("\t\t%s.%s = make(%s, len(%s.%s))\n",
		tgtParam, targetField.Name, targetField.Type, srcParam, sourceField.Name))
	buf.WriteString(fmt.Sprintf("\t\tfor i, src := range %s.%s {\n", srcParam, sourceField.Name))

	for _, mapping := range fieldMappings {
		buf.WriteString(mapping + "\n")
	}

	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")

	return buf.String()
}

// generatePtrSliceFieldMapping generates code for pointer-to-slice fields (e.g., *[]Type)
func generatePtrSliceFieldMapping(sourceField, targetField *FieldInfo, srcParam, tgtParam string, structMap map[string]*StructInfo) string {
	var buf bytes.Buffer

	// For pointer-to-slice, check if pointer is non-nil, then map the slice
	sourceElemType := strings.TrimPrefix(sourceField.BaseType, "[]")
	targetElemType := strings.TrimPrefix(targetField.BaseType, "[]")

	buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", srcParam, sourceField.Name))

	if sourceElemType == targetElemType {
		// Same element types - simple copy
		if targetField.IsPtr {
			buf.WriteString(fmt.Sprintf("\t\tslice := make(%s, len(*%s.%s))\n", sourceField.BaseType, srcParam, sourceField.Name))
			buf.WriteString(fmt.Sprintf("\t\tcopy(slice, *%s.%s)\n", srcParam, sourceField.Name))
			buf.WriteString(fmt.Sprintf("\t\t%s.%s = &slice\n", tgtParam, targetField.Name))
		} else {
			buf.WriteString(fmt.Sprintf("\t\t%s.%s = make(%s, len(*%s.%s))\n", tgtParam, targetField.Name, targetField.Type, srcParam, sourceField.Name))
			buf.WriteString(fmt.Sprintf("\t\tcopy(%s.%s, *%s.%s)\n", tgtParam, targetField.Name, srcParam, sourceField.Name))
		}
	} else {
		// Different element types - need element-wise mapping
		sourceElemStruct := findStruct(sourceElemType, structMap)
		targetElemStruct := findStruct(targetElemType, structMap)

		if sourceElemStruct == nil || targetElemStruct == nil {
			buf.WriteString(fmt.Sprintf("\t\t// TODO: Handle ptr slice field %s manually (element struct not found)\n", sourceField.Name))
		} else {
			targetElemFieldMap := make(map[string]*FieldInfo)
			for i := range targetElemStruct.Fields {
				targetElemFieldMap[targetElemStruct.Fields[i].Name] = &targetElemStruct.Fields[i]
			}

			if targetField.IsPtr {
				buf.WriteString(fmt.Sprintf("\t\tslice := make(%s, len(*%s.%s))\n", targetField.BaseType, srcParam, sourceField.Name))
				buf.WriteString(fmt.Sprintf("\t\tfor i, src := range *%s.%s {\n", srcParam, sourceField.Name))
			} else {
				buf.WriteString(fmt.Sprintf("\t\t%s.%s = make(%s, len(*%s.%s))\n", tgtParam, targetField.Name, targetField.Type, srcParam, sourceField.Name))
				buf.WriteString(fmt.Sprintf("\t\tfor i, src := range *%s.%s {\n", srcParam, sourceField.Name))
			}

			for _, srcField := range sourceElemStruct.Fields {
				tgtField, exists := targetElemFieldMap[srcField.Name]
				if !exists || strings.HasPrefix(srcField.Type, "[]") {
					continue
				}

				tgtRef := fmt.Sprintf("%s.%s[i]", tgtParam, targetField.Name)
				if targetField.IsPtr {
					tgtRef = "slice[i]"
				}

				if tgtField.IsPtr && !srcField.IsPtr {
					buf.WriteString(fmt.Sprintf("\t\t\tval%s := src.%s\n", srcField.Name, srcField.Name))
					buf.WriteString(fmt.Sprintf("\t\t\t%s.%s = &val%s\n", tgtRef, tgtField.Name, srcField.Name))
				} else if !tgtField.IsPtr && srcField.IsPtr {
					buf.WriteString(fmt.Sprintf("\t\t\tif src.%s != nil {\n", srcField.Name))
					buf.WriteString(fmt.Sprintf("\t\t\t\t%s.%s = *src.%s\n", tgtRef, tgtField.Name, srcField.Name))
					buf.WriteString("\t\t\t}\n")
				} else {
					buf.WriteString(fmt.Sprintf("\t\t\t%s.%s = src.%s\n", tgtRef, tgtField.Name, srcField.Name))
				}
			}

			buf.WriteString("\t\t}\n")
			if targetField.IsPtr {
				buf.WriteString(fmt.Sprintf("\t\t%s.%s = &slice\n", tgtParam, targetField.Name))
			}
		}
	}

	buf.WriteString("\t}\n")
	return buf.String()
}

// generatePatchMethodBody generates code for patch methods that return a map with non-nil fields
// Pattern: PatchDbMetadata(req request.UpdateMetadata) map[string]any
func generatePatchMethodBody(method MethodInfo, structMap map[string]*StructInfo) string {
	sourceType := strings.TrimPrefix(method.Params[0].Type, "*")

	sourceStruct := findStruct(sourceType, structMap)

	if sourceStruct == nil {
		return fmt.Sprintf("\t// TODO: Could not find struct definition for %s\n\tpanic(\"not implemented\")", sourceType)
	}

	var buf bytes.Buffer
	reqParam := method.Params[0].Name

	// Create the result map
	buf.WriteString("\tresult := make(map[string]any)\n")

	for _, sourceField := range sourceStruct.Fields {
		// Skip slice fields for now
		if strings.HasPrefix(sourceField.Type, "[]") {
			buf.WriteString(fmt.Sprintf("\t// TODO: Handle slice field %s manually\n", sourceField.Name))
			continue
		}

		// Convert field name to snake_case for database column name
		snakeCaseName := toSnakeCase(sourceField.Name)

		if sourceField.IsPtr {
			// Pointer field: only add to map if non-nil
			buf.WriteString(fmt.Sprintf("\tif %s.%s != nil {\n", reqParam, sourceField.Name))
			buf.WriteString(fmt.Sprintf("\t\tresult[\"%s\"] = *%s.%s\n", snakeCaseName, reqParam, sourceField.Name))
			buf.WriteString("\t}\n")
		} else {
			// Non-pointer field: always add to map
			buf.WriteString(fmt.Sprintf("\tresult[\"%s\"] = %s.%s\n", snakeCaseName, reqParam, sourceField.Name))
		}

	}

	buf.WriteString("\treturn result\n")
	return buf.String()
}

// toSnakeCase converts a CamelCase string to snake_case
// Handles acronyms like ID, SSL, URL properly
func toSnakeCase(s string) string {
	var result bytes.Buffer
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if previous char was lowercase OR if next char is lowercase (end of acronym)
			prevLower := runes[i-1] >= 'a' && runes[i-1] <= 'z'
			nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			if prevLower || nextLower {
				result.WriteByte('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isBuiltinType returns true if the given type name is a Go builtin type
func isBuiltinType(t string) bool {
	builtins := map[string]bool{
		"bool": true, "string": true, "int": true, "int8": true, "int16": true,
		"int32": true, "int64": true, "uint": true, "uint8": true, "uint16": true,
		"uint32": true, "uint64": true, "uintptr": true, "byte": true, "rune": true,
		"float32": true, "float64": true, "complex64": true, "complex128": true,
		"any": true, "error": true,
	}
	return builtins[t]
}

// qualifyType prefixes non-builtin types with their package name
// e.g. qualifyType("DBType", "models") → "models.DBType"
// e.g. qualifyType("*DBType", "models") → "*models.DBType"
// e.g. qualifyType("string", "models") → "string" (builtin, unchanged)
func qualifyType(fieldType, pkg string) string {
	prefix := ""
	base := fieldType

	if strings.HasPrefix(base, "*") {
		prefix = "*"
		base = strings.TrimPrefix(base, "*")
	}

	// Handle slice types
	if strings.HasPrefix(base, "[]") {
		elemType := strings.TrimPrefix(base, "[]")
		if !isBuiltinType(elemType) && !strings.Contains(elemType, ".") {
			return prefix + "[]" + pkg + "." + elemType
		}
		return fieldType
	}

	// Handle map types - don't qualify
	if strings.HasPrefix(base, "map[") {
		return fieldType
	}

	// Simple type: qualify if not builtin and not already qualified
	if !isBuiltinType(base) && !strings.Contains(base, ".") {
		return prefix + pkg + "." + base
	}

	return fieldType
}

// typesCompatible checks if two type strings refer to the same Go type
// (possibly with different package qualification, e.g. "DBType" vs "models.DBType")
func typesCompatible(type1, type2 string) bool {
	if type1 == type2 {
		return true
	}
	parts1 := strings.Split(type1, ".")
	parts2 := strings.Split(type2, ".")
	return parts1[len(parts1)-1] == parts2[len(parts2)-1]
}

// castExpr wraps expr in a type cast to targetType if source and target types differ.
// e.g. castExpr("req.DbType", "string", "models.DBType") → "models.DBType(req.DbType)"
func castExpr(expr, sourceBaseType, targetBaseType string) string {
	if typesCompatible(sourceBaseType, targetBaseType) {
		return expr
	}
	return fmt.Sprintf("%s(%s)", targetBaseType, expr)
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
