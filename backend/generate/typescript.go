package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/efucloud/common"
	"golang.org/x/tools/go/packages"
)

const ident = "  "

func generateTypescriptDefine() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
		return
	}

	dtoDir := path.Join(dir, "pkg", "models", "dtos")

	cfg := &packages.Config{
		Mode: packages.NeedSyntax |
			packages.NeedFiles,
	}

	pkgs, err := packages.Load(
		cfg,
		"./pkg/models/dtos",
	)

	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
		return
	}

	if len(pkgs) == 0 {
		fmt.Println("no packages found:", dtoDir)
		return
	}

	fileStructs := make(map[string][]string)
	allEntries := make(map[string]string)
	allFiles := make(map[string]string)
	fileNeeds := make(map[string][]string)

	for _, pkg := range pkgs {
		for index, item := range pkg.Syntax {

			filename := ""

			if index < len(pkg.GoFiles) {
				n := strings.Split(pkg.GoFiles[index], "/")
				filename = strings.TrimSuffix(n[len(n)-1], ".go")
			}

			if filename == "" {
				continue
			}

			structs := ExtractStructs(item)

			fN := fmt.Sprintf("./%s.d", filename)

			fileStructs[fN] = []string{}

			for s := range structs {
				fileStructs[fN] = append(fileStructs[fN], s)
			}

			sort.Strings(fileStructs[fN])

			content, entries, needImports := GenerateTypescript(structs)

			for _, entry := range entries {
				allEntries[entry] = fN
			}

			for _, need := range needImports {
				if !common.StringInArray(
					need,
					fileNeeds[filename],
				) {
					fileNeeds[filename] = append(
						fileNeeds[filename],
						need,
					)
				}
			}

			allFiles[filename] = content
		}
	}

	generateDir := path.Join(dir, "..", "frontend", "src", "services")

	_ = os.MkdirAll(
		generateDir,
		0o755,
	)

	for n, f := range allFiles {

		needStr := ""

		importFiles := make(map[string][]string)

		if needs := fileNeeds[n]; len(needs) > 0 {

			for _, need := range needs {

				if entryFile, ok := allEntries[need]; ok {

					if !common.StringInArray(
						need,
						importFiles[entryFile],
					) {
						importFiles[entryFile] =
							append(
								importFiles[entryFile],
								need,
							)
					}
				}
			}
		}

		importFileNames := make(
			[]string,
			0,
			len(importFiles),
		)

		for fileName := range importFiles {
			importFileNames = append(
				importFileNames,
				fileName,
			)
		}

		sort.Strings(importFileNames)

		for _, k := range importFileNames {

			if k == fmt.Sprintf(
				"./%s.d",
				n,
			) {
				continue
			}

			v := importFiles[k]

			sort.Strings(v)

			needStr += fmt.Sprintf(
				"import { %s } from '%s';\n",
				strings.Join(v, ", "),
				k,
			)
		}

		if n == "common" {
			needStr = ""
		}

		err = os.WriteFile(
			path.Join(
				generateDir,
				fmt.Sprintf("%s.d.ts", n),
			),
			[]byte(needStr+f),
			0o644,
		)

		if err != nil {
			fmt.Println(err)
		}
	}

	// 外部结构体导入
	converter := NewTypeScriptify()

	converter.CreateType = true
	converter.CreateConstructor = false
	converter.Indent = "    "
	converter.BackupDir = ""
	err = converter.ConvertToFile(
		path.Join(
			generateDir,
			"external.d.ts",
		),
	)

	if err != nil {
		fmt.Printf(
			"converter failed, err: %s",
			err.Error(),
		)
	}

	fileStructs["external.d"] = []string{}

	for _, item := range converter.types {

		allEntries[item] = "./external.d"

		fileStructs["external.d"] =
			append(
				fileStructs["external.d"],
				item,
			)
	}

	data, err := json.Marshal(allEntries)

	if err != nil {
		fmt.Println(err)
		return
	}

	err = os.WriteFile(
		path.Join(
			generateDir,
			"entries.json",
		),
		data,
		0o644,
	)

	if err != nil {
		fmt.Println(err)
	}
}

type StructInformation struct {
	Name        string
	Comments    []string
	Fields      []FieldInformation
	AliasType   string
	NeedImports []string
	TypeParams  []string
}

type FieldInformation struct {
	Name           string
	JsonName       string
	Kind           string
	Comments       []string
	IsArray        bool
	Required       bool
	EnumValues     []interface{}
	Default        string
	Enum           string
	Length         string
	EmbedStruct    interface{}
	TypeScriptType string
	NeedImports    []string
	Inline         bool
	InlineFields   []FieldInformation
}

var kinds map[string]string

func init() {
	kinds = make(map[string]string)
	kinds[reflect.Bool.String()] = "boolean"
	kinds[reflect.Interface.String()] = "any"
	kinds[reflect.Int.String()] = "number"
	kinds[reflect.Int8.String()] = "number"
	kinds[reflect.Int16.String()] = "number"
	kinds[reflect.Int32.String()] = "number"
	kinds[reflect.Int64.String()] = "number"
	kinds[reflect.Uint.String()] = "number"
	kinds[reflect.Uint8.String()] = "number"
	kinds[reflect.Uint16.String()] = "number"
	kinds[reflect.Uint32.String()] = "number"
	kinds[reflect.Uint64.String()] = "number"
	kinds[reflect.Float32.String()] = "number"
	kinds[reflect.Float64.String()] = "number"
	kinds[reflect.String.String()] = "string"
	kinds["ArrayFloat64"] = "Float64Array"
	kinds["ArrayString"] = "string[]"
	kinds["ArrayUint"] = "number[]"
	kinds["time"] = "string"
	kinds["PatchSubsetValue"] = "PatchSubsetValue"
	kinds["ClusterServerGroupChecks"] = "ClusterServerGroupCheck[]"
	kinds["ApplicationKubernetesResources"] = "ApplicationKubernetesResource[]"
}
func getKind(kind string) string {
	if k, ex := kinds[kind]; ex {
		return k
	}
	return ""
}
func getDefaultKind(unresolved map[token.Pos]*ast.Ident, pos token.Pos) string {
	id := unresolved[pos]
	if id != nil {
		return id.String()
	}
	return ""
}
func getComment(commentGroups []*ast.CommentGroup, pos token.Pos) (comments []string) {
	var (
		bigest token.Pos
	)
	for i := 0; i < len(commentGroups); i++ {
		item := commentGroups[i]
		if item.Pos() >= bigest {
			if item.Pos() <= pos {
				bigest = item.Pos()
				comments = []string{}
				for _, i := range item.List {
					t := strings.TrimSpace(i.Text)
					if len(t) > 0 && t != "//" {
						comments = append(comments, t)
					}
				}
			}
		}
	}
	return comments
}

func appendUniqueStrings(values []string, adds ...string) []string {
	for _, item := range adds {
		if len(strings.TrimSpace(item)) == 0 {
			continue
		}
		if !common.StringInArray(item, values) {
			values = append(values, item)
		}
	}
	return values
}

func typeNameFromExpr(expr ast.Expr) string {
	switch s := expr.(type) {
	case *ast.Ident:
		return s.Name
	case *ast.SelectorExpr:
		return s.Sel.Name
	case *ast.StarExpr:
		return typeNameFromExpr(s.X)
	case *ast.ArrayType:
		return typeNameFromExpr(s.Elt)
	case *ast.IndexExpr:
		return typeNameFromExpr(s.X)
	case *ast.IndexListExpr:
		return typeNameFromExpr(s.X)
	default:
		return ""
	}
}

func extractTypeParams(typeParams *ast.FieldList) (names []string, values map[string]struct{}) {
	values = make(map[string]struct{})
	if typeParams == nil {
		return names, values
	}
	for _, field := range typeParams.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
			values[name.Name] = struct{}{}
		}
	}
	return names, values
}

func normalizeTSTypeName(name string) string {
	if len(name) == 0 {
		return ""
	}
	if kind := getKind(name); len(kind) > 0 {
		return kind
	}
	switch name {
	case "any":
		return "any"
	case "interface{}":
		return "any"
	case "Time":
		return "string"
	default:
		return name
	}
}

func resolveMapKeyType(expr ast.Expr) string {
	switch s := expr.(type) {
	case *ast.Ident:
		switch s.Name {
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
			return "number"
		default:
			return "string"
		}
	default:
		return "string"
	}
}

func renderInlineStructType(fields []FieldInformation) string {
	if len(fields) == 0 {
		return "Record<string, any>"
	}
	content := "{ "
	for _, field := range fields {
		if field.Inline {
			if len(field.InlineFields) > 0 {
				inlineType := strings.TrimSpace(renderInlineStructType(field.InlineFields))
				inlineType = strings.TrimPrefix(inlineType, "{")
				inlineType = strings.TrimSuffix(inlineType, "}")
				content += inlineType + " "
				continue
			}
			content += fmt.Sprintf("& %s ", field.TypeScriptType)
			continue
		}
		if len(strings.TrimSpace(field.JsonName)) == 0 {
			continue
		}
		content += field.JsonName
		if !field.Required {
			content += "?"
		}
		fieldType, _ := renderFieldType(field)
		content += fmt.Sprintf(": %s; ", fieldType)
	}
	content += "}"
	return content
}

func resolveTypeExpr(expr ast.Expr, fileUnresolveds map[token.Pos]*ast.Ident, typeParams map[string]struct{}) (tsType string, needImports []string, inlineFields []FieldInformation) {
	switch s := expr.(type) {
	case *ast.Ident:
		if _, ok := typeParams[s.Name]; ok {
			return s.Name, nil, nil
		}
		tsType = normalizeTSTypeName(s.Name)
		if tsType == s.Name && getKind(s.Name) == "" && tsType != "any" {
			needImports = appendUniqueStrings(needImports, s.Name)
		}
	case *ast.SelectorExpr:
		pkgName := typeNameFromExpr(s.X)
		switch pkgName {
		case "time":
			tsType = "string"
		case "gorm":
			switch s.Sel.Name {
			case "DeletedAt":
				tsType = "string"
			default:
				tsType = s.Sel.Name
				needImports = appendUniqueStrings(needImports, s.Sel.Name)
			}
		case "metav1":
			switch s.Sel.Name {
			case "Time", "MicroTime":
				tsType = "string"
			default:
				tsType = s.Sel.Name
				needImports = appendUniqueStrings(needImports, s.Sel.Name)
			}
		default:
			tsType = s.Sel.Name
			needImports = appendUniqueStrings(needImports, s.Sel.Name)
		}
	case *ast.StarExpr:
		return resolveTypeExpr(s.X, fileUnresolveds, typeParams)
	case *ast.ArrayType:
		var innerType string
		innerType, needImports, inlineFields = resolveTypeExpr(s.Elt, fileUnresolveds, typeParams)
		if len(innerType) == 0 {
			innerType = "any"
		}
		tsType = fmt.Sprintf("%s[]", innerType)
	case *ast.IndexExpr:
		var baseType, paramType string
		baseType, needImports, _ = resolveTypeExpr(s.X, fileUnresolveds, typeParams)
		paramType, paramImports, _ := resolveTypeExpr(s.Index, fileUnresolveds, typeParams)
		needImports = appendUniqueStrings(needImports, paramImports...)
		tsType = fmt.Sprintf("%s<%s>", baseType, paramType)
	case *ast.IndexListExpr:
		var baseType string
		baseType, needImports, _ = resolveTypeExpr(s.X, fileUnresolveds, typeParams)
		var paramTypes []string
		for _, index := range s.Indices {
			paramType, paramImports, _ := resolveTypeExpr(index, fileUnresolveds, typeParams)
			paramTypes = append(paramTypes, paramType)
			needImports = appendUniqueStrings(needImports, paramImports...)
		}
		tsType = fmt.Sprintf("%s<%s>", baseType, strings.Join(paramTypes, ", "))
	case *ast.InterfaceType:
		tsType = "any"
	case *ast.MapType:
		var valueType string
		valueType, needImports, inlineFields = resolveTypeExpr(s.Value, fileUnresolveds, typeParams)
		if len(valueType) == 0 {
			valueType = "any"
		}
		tsType = fmt.Sprintf("Record<%s, %s>", resolveMapKeyType(s.Key), valueType)
	case *ast.StructType:
		inlineFields = extractFieldsFromStructType(s, fileUnresolveds, typeParams)
		tsType = renderInlineStructType(inlineFields)
	default:
		if name := getDefaultKind(fileUnresolveds, expr.Pos()); len(name) > 0 {
			if _, ok := typeParams[name]; ok {
				return name, nil, nil
			}
			tsType = normalizeTSTypeName(name)
			if tsType == name && getKind(name) == "" && tsType != "any" {
				needImports = appendUniqueStrings(needImports, name)
			}
			return tsType, needImports, inlineFields
		}
		fmt.Printf("unknown type expression: %T\n", expr)
		tsType = "any"
	}
	return tsType, needImports, inlineFields
}

func parseFieldMetadata(field *ast.Field, fi *FieldInformation) (tag reflect.StructTag, rawJSON string, skip bool) {
	if field.Tag != nil {
		tag = reflect.StructTag(strings.TrimPrefix(strings.TrimSuffix(field.Tag.Value, "`"), "`"))
		rawJSON = tag.Get("json")
	}
	if rawJSON == "-" {
		return tag, rawJSON, true
	}
	fi.Required = len(tag.Get("validate")) != 0 || fi.Name == "ID" || fi.Name == "CreatedAt" || fi.Name == "UpdatedAt"
	gorm := tag.Get("gorm")
	if strings.Contains(gorm, "primarykey") {
		fi.Required = true
	}
	if len(gorm) > 0 {
		for _, it := range strings.Split(gorm, ";") {
			sp := strings.Split(it, ":")
			if len(sp) != 2 {
				continue
			}
			switch sp[0] {
			case "default":
				fi.Default = sp[1]
			case "type":
				if strings.HasPrefix(sp[1], "varchar(") {
					fi.Length = strings.TrimSuffix(strings.TrimPrefix(sp[1], "varchar("), ")")
				}
			}
		}
	}
	return tag, rawJSON, false
}

func assignResolvedFieldType(fi *FieldInformation, expr ast.Expr, fileUnresolveds map[token.Pos]*ast.Ident, typeParams map[string]struct{}) {
	fi.Kind = getDefaultKind(fileUnresolveds, expr.Pos())
	tsType, needImports, inlineFields := resolveTypeExpr(expr, fileUnresolveds, typeParams)
	fi.TypeScriptType = tsType
	fi.NeedImports = appendUniqueStrings(fi.NeedImports, needImports...)
	fi.InlineFields = inlineFields
}

func extractFieldsFromStructType(structType *ast.StructType, fileUnresolveds map[token.Pos]*ast.Ident, typeParams map[string]struct{}) (fields []FieldInformation) {
	for _, field := range structType.Fields.List {
		var comments []string
		if field.Doc != nil {
			for _, c := range field.Doc.List {
				comments = append(comments, c.Text)
			}
		}
		if len(field.Names) == 0 {
			fi := FieldInformation{
				Name:     typeNameFromExpr(field.Type),
				Comments: comments,
			}
			tag, rawJSON, skip := parseFieldMetadata(field, &fi)
			if skip {
				continue
			}
			assignResolvedFieldType(&fi, field.Type, fileUnresolveds, typeParams)
			jsonName := rawJSON
			if strings.Contains(jsonName, ",") {
				jsonName = strings.Split(jsonName, ",")[0]
			}
			fi.Inline = rawJSON == "" || rawJSON == ",inline"
			if fi.Inline {
				fi.JsonName = ",inline"
			} else if len(jsonName) > 0 {
				fi.JsonName = jsonName
			} else {
				fi.JsonName = fi.Name
			}
			if len(tag.Get("enum")) > 0 {
				fi.Enum = tag.Get("enum")
			}
			fields = append(fields, fi)
			continue
		}
		if field.Tag == nil {
			continue
		}
		for _, name := range field.Names {
			fi := FieldInformation{
				Name:     name.Name,
				Comments: comments,
			}
			tag, rawJSON, skip := parseFieldMetadata(field, &fi)
			if skip {
				continue
			}
			jsonName := rawJSON
			if jsonName == "" || jsonName == ".inline" {
				jsonName = fi.Name
			}
			if strings.Contains(jsonName, ",") {
				jsonName = strings.Split(jsonName, ",")[0]
			}
			fi.JsonName = jsonName
			assignResolvedFieldType(&fi, field.Type, fileUnresolveds, typeParams)
			if len(tag.Get("enum")) > 0 {
				fi.Enum = tag.Get("enum")
			}
			fields = append(fields, fi)
		}
	}
	return fields
}

func renderFieldType(field FieldInformation) (string, []string) {
	if len(strings.TrimSpace(field.TypeScriptType)) > 0 {
		return field.TypeScriptType, field.NeedImports
	}
	if len(field.Kind) == 0 {
		return "any", field.NeedImports
	}
	if field.Kind == "DeletedAt" {
		return "string", field.NeedImports
	}
	if field.Kind == "PatchSubsetValues" {
		field.Kind = "PatchSubsetValue[]"
	}
	kind := getKind(field.Kind)
	if len(kind) == 0 {
		if field.IsArray {
			return fmt.Sprintf("%s[]", field.Kind), appendUniqueStrings(field.NeedImports, field.Kind)
		}
		return field.Kind, appendUniqueStrings(field.NeedImports, field.Kind)
	}
	if field.IsArray {
		return fmt.Sprintf("%s[]", kind), field.NeedImports
	}
	return kind, field.NeedImports
}

func renderField(field FieldInformation) (content string, needImports []string) {
	if len(strings.TrimSpace(field.JsonName)) == 0 {
		return "", nil
	}
	for _, i := range field.Comments {
		t := strings.TrimSuffix(strings.TrimPrefix(i, "//"), "\n")
		t = strings.TrimSpace(t)
		if len(t) > 0 {
			content += fmt.Sprintf("%s%s\n", ident, i)
		}
	}
	if len(field.Default) > 0 {
		content += fmt.Sprintf("%s//默认值: %s\n", ident, field.Default)
	}
	if len(field.EnumValues) > 0 {
		var ev []string
		for _, i := range field.EnumValues {
			ev = append(ev, fmt.Sprintf("%s//%v", ident, i))
		}
		content += fmt.Sprintf("%s//可选值: %s\n", ident, strings.Join(ev, ");"))
	}
	if len(field.Length) > 0 {
		content += fmt.Sprintf("%s//最大长度: %s\n", ident, field.Length)
	}
	fieldType, imports := renderFieldType(field)
	content += fmt.Sprintf("%s%s", ident, field.JsonName)
	if !field.Required {
		content += "?"
	}
	content += fmt.Sprintf(": %s;\n", fieldType)
	return content, imports
}

func GenerateTypescript(structs map[string]*StructInformation) (content string, entries []string, needImports []string) {
	for _, item := range structs {
		entries = append(entries, item.Name)
	}
	sort.Strings(entries)
	for _, key := range entries {
		item := structs[key]
		for _, c := range item.Comments {
			content += fmt.Sprintf("%s\n", c)
		}
		if item.Name == "KubernetesResource" {
			content += `export type KubernetesResource = {
  apiVersion?: string;
  kind?: string;
  metadata: {
    name: string;
    namespace?: string;
    uid?: string;
    [key: string]: any;
  };
  spec?: any;
  status?: any;
  [key: string]: any;
};
export type WatchEventType = 'ADDED' | 'MODIFIED' | 'DELETED' | 'BOOKMARK';

export interface WatchEvent<T extends KubernetesResource = KubernetesResource> {
  Type: WatchEventType;
  Object: T;
}
`
			continue
		}
		if len(strings.TrimSpace(item.AliasType)) > 0 {
			typeParams := ""
			if len(item.TypeParams) > 0 {
				typeParams = fmt.Sprintf("<%s>", strings.Join(item.TypeParams, ", "))
			}
			content += fmt.Sprintf("export type %s%s = %s;\n", item.Name, typeParams, item.AliasType)
			needImports = appendUniqueStrings(needImports, item.NeedImports...)
			continue
		}
		typeParams := ""
		if len(item.TypeParams) > 0 {
			var definitions []string
			for _, typeParam := range item.TypeParams {
				definitions = append(definitions, fmt.Sprintf("%s = any", typeParam))
			}
			typeParams = fmt.Sprintf("<%s>", strings.Join(definitions, ", "))
		}
		content += fmt.Sprintf("export type %s%s = { \n", item.Name, typeParams)
		merged := ""
		for _, field := range item.Fields {
			if field.Inline {
				if len(field.InlineFields) > 0 {
					for _, inlineField := range field.InlineFields {
						inlineContent, inlineImports := renderField(inlineField)
						content += inlineContent
						needImports = appendUniqueStrings(needImports, inlineImports...)
					}
					continue
				}
				fieldType, imports := renderFieldType(field)
				if len(strings.TrimSpace(fieldType)) == 0 {
					fieldType = "any"
				}
				merged += fmt.Sprintf(" & %s", fieldType)
				needImports = appendUniqueStrings(needImports, imports...)
				continue
			}
			fieldContent, imports := renderField(field)
			content += fieldContent
			needImports = appendUniqueStrings(needImports, imports...)
		}
		content += fmt.Sprintf("}%s; \n", merged)
	}
	return content, entries, needImports
}
func ExtractStructs(item *ast.File) (structs map[string]*StructInformation) {
	structs = make(map[string]*StructInformation)
	fileComments := item.Comments
	fileUnresolveds := make(map[token.Pos]*ast.Ident)
	for _, it := range item.Unresolved {
		fileUnresolveds[it.Pos()] = it
	}
	for n, obj := range item.Scope.Objects {
		if n == "ArrayString" || n == "ArrayUint" {
			continue
		}
		if obj.Kind.String() != "type" {
			continue
		}
		o := obj.Decl.(*ast.TypeSpec)
		info := &StructInformation{
			Name:     n,
			Comments: getComment(fileComments, o.Name.Pos()),
		}
		typeParamNames, typeParamSet := extractTypeParams(o.TypeParams)
		info.TypeParams = typeParamNames
		switch s := o.Type.(type) {
		case *ast.StructType:
			info.Fields = extractFieldsFromStructType(s, fileUnresolveds, typeParamSet)
		default:
			tsType, needImports, _ := resolveTypeExpr(o.Type, fileUnresolveds, typeParamSet)
			if n == "AnyJsonData" && tsType == "Record<string, any>" {
				tsType = "Record<string, unknown>"
			}
			info.AliasType = tsType
			info.NeedImports = appendUniqueStrings(info.NeedImports, needImports...)
		}
		structs[n] = info
	}
	return structs
}
