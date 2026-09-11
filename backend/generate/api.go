package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

const request = "import { request } from '@umijs/max';\n"
const SystemApiName = "SystemApiName"
const SkipFrontAPIName = "-"

type RestAPI struct {
	routes        []restful.Route
	apis          map[string]ApiData
	systemApiName string
	files         map[string]string
	ignores       map[string]string
	entries       map[string]string
	entrySource   map[string]string
	entryArray    map[string][]string
	fileFunctions map[string][]string
}

type ApiData struct {
	DocumentName   string
	RequestModel   string
	ResponseModel  string
	Name           string
	Doc            string
	Notes          string
	Path           string
	ContentType    []string
	Method         string
	Parameters     map[string]Parameters
	Response       map[int]string
	HasRequestBody bool
	HasPathParams  bool
	HasQueryParams bool
	HasParams      bool
	PathParams     []string
	FormParams     []string
}

func (api ApiData) String() string {
	return api.Method + " " + api.Path
}

func (api ApiData) parameterNames() []string {
	names := make([]string, 0, len(api.Parameters))
	for name := range api.Parameters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (api ApiData) GenerateAPI() (content string, entries, functions []string) {
	content = fmt.Sprintf("//%s\n", api.Doc)
	content += fmt.Sprintf("//%s\n", api.Notes)
	content += fmt.Sprintf("//请求方法: %s\n", api.Method)
	content += fmt.Sprintf("//请求地址: %s\n", api.Path)
	pNames := api.parameterNames()
	for _, name := range pNames {
		item := api.Parameters[name]
		if item.DataType == "integer" {
			item.DataType = "number"
		}
		if item.Position == "path" {
			api.Path = strings.ReplaceAll(api.Path, fmt.Sprintf("{%s}", item.Name), fmt.Sprintf("${%s}", item.Name))
			api.Path = strings.ReplaceAll(api.Path, fmt.Sprintf("{%s:*}", item.Name), fmt.Sprintf("${%s}", item.Name))
			api.HasParams = true
			api.HasPathParams = true
			api.PathParams = append(api.PathParams, item.Name)
			content += fmt.Sprintf("//参数名: %s 参数类型: %s 参数位置: %s 是否必须: %t  参数说明: %s\n", item.Name, item.DataType, item.Position, item.Required, item.Description)
		}
		if item.Position == "query" {
			api.HasParams = true
			api.HasQueryParams = true
			content += fmt.Sprintf("//参数名: %s 参数类型: %s 参数位置: %s 是否必须: %t  参数说明: %s\n", item.Name, item.DataType, item.Position, item.Required, item.Description)
		}
		if item.Position == "form" {
			api.FormParams = append(api.FormParams, item.Name)
		}
		api.Parameters[name] = item

	}
	content += fmt.Sprintf("export async function %s", api.Name)
	functions = append(functions, api.Name)

	if len(api.ResponseModel) > 0 && api.ResponseModel != "string" {
		entries = append(entries, api.ResponseModel)
	}

	content += "("
	if len(api.Parameters) > 0 {
		content += api.Params()
		if api.HasRequestBody {
			if len(api.RequestModel) > 0 {
				content += fmt.Sprintf("  data: %s, ", api.RequestModel)
			} else {
				content += fmt.Sprintf("  data: %s, ", "any")
			}
			entries = append(entries, api.RequestModel)
		}
		content += "  options?: { [key: string]: any }) {\n"
	} else {
		content += "  options?: { [key: string]: any }) {\n"

	}

	if api.HasPathParams || len(api.FormParams) > 0 {
		p := api.PathParams
		p = append(p, api.FormParams...)
		content += fmt.Sprintf("  const { %s, ...rest } = params;\n", strings.Join(p, ", "))
	}
	if len(api.FormParams) > 0 {
		if common.StringInArray(config.RequestForm, api.ContentType) {
			content += "  const formData = new URLSearchParams();\n"
		} else {
			content += "  const formData = new FormData();\n"
		}
		for _, name := range api.FormParams {
			content += fmt.Sprintf("  formData.append('%s', %s);\n", name, name)
		}
	}
	switch api.ResponseModel {
	case "ArrayString":
		api.ResponseModel = "string[]"
	case "ArrayUint":
		api.ResponseModel = "number[]"
	}
	if len(api.ResponseModel) > 0 && api.ResponseModel != "string" {
		content += fmt.Sprintf("  return request<%s>(`%s`, {\n", api.ResponseModel, api.Path)

	} else {
		content += fmt.Sprintf("  return request(`%s`, {\n", api.Path)
	}
	content += fmt.Sprintf("    method: '%s',\n", api.Method)
	//请求头
	if len(api.ContentType) == 0 {
		api.ContentType = []string{restful.MIME_JSON}
	}
	content += "    headers: {\n"
	content += fmt.Sprintf("      'Content-Type': '%s',\n", strings.Join(api.ContentType, ";"))
	content += "    },\n"
	if strings.Contains(strings.Join(api.ContentType, ";"), "application/json-patch+json") {
		content += "    data,\n"
	}
	if len(api.RequestModel) > 0 {
		content += "    data,\n"
	} else if len(api.FormParams) > 0 {
		if common.StringInArray(config.RequestForm, api.ContentType) {
			content += "    data: formData.toString(),\n"
		} else {
			content += "    data: formData,\n"
		}
	}
	if api.HasPathParams {
		content += "    params: { ...rest },\n"
	} else if api.HasQueryParams {
		content += "    params: params,\n"
	}

	content += "    ...(options || {}),\n  });\n}\n"
	return content, entries, functions
}
func (api ApiData) Params() string {
	content := ""
	if api.HasQueryParams || api.HasPathParams || len(api.FormParams) > 0 {
		content += "\n  params: {\n"
		pathParam := ""
		formParam := ""
		queryParam := ""
		for _, name := range api.parameterNames() {
			item := api.Parameters[name]
			switch item.Position {
			case "path":
				pathParam += fmt.Sprintf("    %s: %s;// %s\n", item.Name, item.DataType, item.Description)
			case "query":
				require := "?"
				if item.Required {
					require = ""
				}
				//if item.Name == "ids" {
				//	queryParam += fmt.Sprintf("    %s%s: string;//%s\n", item.Name, require, item.Description)
				//} else if item.Name == "uuids" {
				//	queryParam += fmt.Sprintf("    %s%s: string;// %s\n", item.Name, require, item.Description)
				//} else {
				//	queryParam += fmt.Sprintf("    %s%s:%s;// %s\n", item.Name, require, item.DataType, item.Description)
				//}
				queryParam += fmt.Sprintf("    %s%s: %s;// %s\n", item.Name, require, item.DataType, item.Description)
			case "form":
				formParam += fmt.Sprintf("    %s: %s;// %s\n", item.Name, item.DataType, item.Description)
			case "multipart/form-data":
				fmt.Println("unsupported multipart/form-data")
			}
		}
		content += pathParam
		content += formParam
		content += queryParam
		content += "  },\n"
	}
	return content
}

type Parameters struct {
	Name        string //参数名
	DataType    string
	Position    string //query path
	Description string
	Required    bool
	Enum        []string
	Default     string
	Kind        int
}

func NewRestAPI(frontApiName string) *RestAPI {
	api := &RestAPI{
		apis:          make(map[string]ApiData),
		systemApiName: frontApiName,
		files:         make(map[string]string),
		ignores:       make(map[string]string),
		entries:       make(map[string]string),
		entrySource:   make(map[string]string),
		entryArray:    make(map[string][]string),
		fileFunctions: make(map[string][]string),
	}
	if len(api.systemApiName) == 0 {
		api.systemApiName = SystemApiName
	}
	return api
}
func (rest *RestAPI) AddRoute(route restful.Route) {
	rest.routes = append(rest.routes, route)
}
func (rest *RestAPI) AddStructIgnores(ignores ...string) {
	for _, item := range ignores {
		rest.ignores[item] = item
	}
}

func GetStructFieldDescription(item reflect.Type) string {
	result, _ := json.Marshal(ExtractStructFieldDescription(item))
	return string(result)
}
func ExtractStructFieldDescription(item reflect.Type) (result map[string]string) {
	result = make(map[string]string)
	for i := 0; i < item.NumField(); i++ {
		jsonName := item.Field(i).Tag.Get("json")
		if len(jsonName) == 0 {
			jsonName = item.Field(i).Name
		}
		description := item.Field(i).Tag.Get("description")
		if len(description) == 0 {
			description = item.Field(i).Name
		}
		result[jsonName] = description
	}
	return
}

func (rest *RestAPI) GenerateToDir(dir string) {
	rest.Generate()
	_ = os.MkdirAll(dir, 0o755)
	if data, err := os.ReadFile(path.Join(dir, "entries.json")); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &rest.entrySource)
	}
	for k, c := range rest.files {
		//生成依type导入
		fileMap := make(map[string][]string)
		if arr, exist := rest.entryArray[k]; exist {
			for _, item := range arr {
				if key, ok := rest.entrySource[item]; ok {
					if !common.StringInArray(item, fileMap[key]) {
						fileMap[key] = append(fileMap[key], item)
					}
				}
			}
		}
		imports := ""
		importFiles := make([]string, 0, len(fileMap))
		for fileName := range fileMap {
			importFiles = append(importFiles, fileName)
		}
		sort.Strings(importFiles)
		for _, n := range importFiles {
			l := fileMap[n]
			sort.Strings(l)
			imports += fmt.Sprintf("import { %s } from '%s';\n", strings.Join(l, ", "), n)
		}
		_ = os.WriteFile(path.Join(dir, fmt.Sprintf("%s.api.ts", k)), []byte(request+"\n"+imports+"\n"+c), 0o644)
	}
	//exportContent := ""
	//for k, v := range rest.fileFunctions {
	//	exportContent += fmt.Sprintf("import { %s }  from './%s.api';\n", strings.Join(v, ","), k)
	//}
	//exportContent += "\n//导出\nexport {\n"
	//for _, v := range rest.fileFunctions {
	//	exportContent += fmt.Sprintf("  %s,\n", strings.Join(v, ","))
	//}
	//exportContent += "};"
	//_ = os.WriteFile(path.Join(dir, "service.api.ts"), []byte(exportContent), os.ModePerm)
	//_ = os.RemoveAll(path.Join(dir, "entries.json"))
}
func (rest *RestAPI) Generate() {

	rest.ParserRoutes()
	var names []string
	for _, api := range rest.apis {
		names = append(names, api.String())
	}
	sort.Strings(names)
	for _, apiName := range names {

		api := rest.apis[apiName]

		text, entries, functions := api.GenerateAPI()
		rest.files[api.DocumentName] += text
		for _, entry := range entries {
			if !common.StringInArray(entry, rest.entryArray[api.DocumentName]) {
				rest.entryArray[api.DocumentName] = append(rest.entryArray[api.DocumentName], entry)
			}
		}
		rest.fileFunctions[api.DocumentName] = functions
	}
}
func (rest *RestAPI) ParserRoutes() {
	allRoutes := make(map[string]restful.Route)
	var names []string
	for _, route := range rest.routes {
		names = append(names, route.String())
		allRoutes[route.String()] = route
	}
	sort.Strings(names)
	for _, routeName := range names {
		route := allRoutes[routeName]
		var api ApiData
		api.Parameters = make(map[string]Parameters)
		api.Response = make(map[int]string)
		api.ContentType = route.Consumes
		api.Doc = route.Doc
		api.Notes = route.Notes
		api.Path = route.Path
		api.Method = route.Method
		if name, exist := route.Metadata[rest.systemApiName]; exist {
			api.Name = fmt.Sprintf("%v", name)
			api.Name = strings.ReplaceAll(api.Name, "[", "")
			api.Name = strings.ReplaceAll(api.Name, "]", "")
			if api.Name == SkipFrontAPIName || len(strings.TrimSpace(api.Name)) == 0 {
				continue
			}
		} else {
			//todo 驼峰
			api.Name = fmt.Sprintf("%s%s", strings.ToLower(route.Method), route.Operation)
		}
		if doc, ex := route.Metadata[restfulspec.KeyOpenAPITags]; ex {
			n := strings.ReplaceAll(fmt.Sprintf("%v", doc), "[", "")
			n = strings.ReplaceAll(n, "]", "")
			api.DocumentName = strings.ReplaceAll(n, "-", "_")
		} else {
			api.DocumentName = "api"
		}
		for _, param := range route.ParameterDocs {
			var p Parameters
			p.Description = param.Data().Description
			p.Name = param.Data().Name
			p.DataType = param.Data().DataType
			p.Required = param.Data().Required
			p.Default = param.Data().DefaultValue
			p.Enum = param.Data().PossibleValues
			switch param.Kind() {
			case restful.PathParameterKind:
				p.Position = "path"
			case restful.QueryParameterKind:
				p.Position = "query"
			case restful.BodyParameterKind:
				p.Position = "body"
				api.HasRequestBody = true
				if strings.Contains(p.DataType, ".") {
					dt := strings.Split(p.DataType, ".")
					if len(dt) >= 2 {
						p.DataType = dt[len(dt)-1]
					}
				}
			case restful.HeaderParameterKind:
				p.Position = "header"
			case restful.FormParameterKind:
				p.Position = "form"
			case restful.MultiPartFormParameterKind:
				p.Position = "multipart/form-data"
			default:
				continue
			}
			api.Parameters[p.Name] = p
			if route.ReadSample != nil {
				read := reflect.ValueOf(route.ReadSample).Type()
				if read.Kind() != reflect.String {
					k := strings.TrimPrefix(read.Name(), "*")
					if strings.Contains(k, ".") {
						sp := strings.Split(k, ".")
						spLen := len(sp)
						k = sp[spLen-1]
					}
					api.RequestModel = k
				}
			}
			if route.WriteSample != nil {
				write := reflect.ValueOf(route.WriteSample).Type()
				if write.Kind() != reflect.String {
					if write.Kind() != reflect.Slice {
						k := strings.TrimPrefix(write.Name(), "*")
						if strings.Contains(k, ".") {
							sp := strings.Split(k, ".")
							spLen := len(sp)
							k = sp[spLen-1]
						}
						api.ResponseModel = k
					}
				}
			}
			for _, res := range route.ResponseErrors {
				if res.Code == http.StatusOK {
					if res.Model != nil {
						successRes := reflect.ValueOf(res.Model).Type()
						if len(api.ResponseModel) == 0 {
							api.ResponseModel = successRes.Name()
						}
						if successRes.Kind().String() == reflect.Pointer.String() || successRes.Kind().String() == reflect.Struct.String() {
							api.Response[res.Code] = successRes.Name()
						}
					}
				}
			}
		}
		rest.apis[api.String()] = api
	}
}
