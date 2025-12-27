package LingEcho

import (
	_ "embed"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/code-100-precent/LingFramework/pkg/constants"
	docsPkg "github.com/code-100-precent/LingFramework/pkg/docs"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

//go:embed  templates/apidocs.html
var apiDocHTML string

type OptionFunc func(*UriDoc)

const (
	TYPE_DATE    = "date"
	TYPE_STRING  = "string"
	TYPE_INT     = "int"
	TYPE_FLOAT   = "float"
	TYPE_BOOLEAN = "boolean"
	TYPE_OBJECT  = "object"
	TYPE_MAP     = "map"
)

type DocField struct {
	FieldName string     `json:"-"`
	Name      string     `json:"name"`
	Desc      string     `json:"desc,omitempty"`
	Type      string     `json:"type,omitempty"`
	Default   any        `json:"default,omitempty"`
	Required  bool       `json:"required,omitempty"`
	CanNull   bool       `json:"canNull,omitempty"`
	IsArray   bool       `json:"isArray,omitempty"`
	IsPrimary bool       `json:"isPrimary,omitempty"`
	Fields    []DocField `json:"fields,omitempty"`
}

type WebObjectDoc struct {
	Group        string     `json:"group"`
	Path         string     `json:"path"`
	Desc         string     `json:"desc,omitempty"`
	AuthRequired bool       `json:"authRequired,omitempty"`
	AllowMethods []string   `json:"allowMethods,omitempty"`
	Fields       []DocField `json:"fields,omitempty"` // Request Body
	Filters      []string   `json:"filters,omitempty"`
	Orders       []string   `json:"orders,omitempty"`
	Searches     []string   `json:"searches,omitempty"`
	Editables    []string   `json:"editables,omitempty"`
	Views        []UriDoc   `json:"views,omitempty"`
}

type UriDoc struct {
	MethodRef    any       `json:"-"` // just for quick jump to method
	Group        string    `json:"group"`
	Path         string    `json:"path"`
	Summary      string    `json:"summary"`
	Desc         string    `json:"desc,omitempty"`
	AuthRequired bool      `json:"authRequired,omitempty"`
	Method       string    `json:"method"` // "GET" "POST" "DELETE" "PUT" "PATCH"
	Request      *DocField `json:"request"`
	Response     *DocField `json:"response"`
}

func RegisterHandler(prefix string, r *gin.Engine, uriDocs []UriDoc, objDocs []WebObjectDoc, db *gorm.DB) {
	RegisterHandlerWithAutoGen(prefix, r, uriDocs, objDocs, db, true)
}

// RegisterHandlerWithAutoGen 注册文档处理器，支持自动生成
func RegisterHandlerWithAutoGen(prefix string, r *gin.Engine, uriDocs []UriDoc, objDocs []WebObjectDoc, db *gorm.DB, autoGen bool) {
	prefix = strings.TrimSuffix(prefix, "/")

	// 如果启用自动生成，尝试从路由中提取文档
	if autoGen {
		collector := docsPkg.NewRouteCollector()
		generator := docsPkg.NewAutoDocGenerator(collector)
		autoUriDocs := generator.GenerateUriDocs(r)

		// 合并手动定义的文档和自动生成的文档
		// 手动定义的文档优先，自动生成的文档只补充缺失的
		uriDocsMap := make(map[string]UriDoc)
		// 先添加手动定义的文档（这些已经有完整的 Desc、Request、Response 等信息）
		for _, doc := range uriDocs {
			key := doc.Method + ":" + doc.Path
			uriDocsMap[key] = doc
		}
		// 转换并合并自动生成的文档（只添加手动定义中没有的）
		for _, doc := range autoUriDocs {
			key := doc.Method + ":" + doc.Path
			if _, exists := uriDocsMap[key]; !exists {
				// 自动生成的文档，使用智能描述
				uriDocsMap[key] = convertDocsUriDoc(doc)
			}
			// 如果已存在手动定义的文档，保留手动定义的（包括 Desc），不覆盖
		}
		// 转换回切片
		uriDocs = make([]UriDoc, 0, len(uriDocsMap))
		for _, doc := range uriDocsMap {
			uriDocs = append(uriDocs, doc)
		}
	}

	r.GET(prefix+".json", func(ctx *gin.Context) {
		ctx.Set(constants.DbField, db)
		docs := map[string]any{
			"uris": uriDocs,
			"objs": objDocs,
			"site": GetRenderPageContext(ctx),
		}
		ctx.JSON(http.StatusOK, docs)
	})

	// OpenAPI 导出
	r.GET(prefix+"/openapi.json", func(ctx *gin.Context) {
		scheme := "http"
		if ctx.Request.TLS != nil {
			scheme = "https"
		}
		baseURL := scheme + "://" + ctx.Request.Host
		openapiGen := docsPkg.NewOpenAPIGenerator(baseURL, "1.0.0", "LingFramework API")
		// 转换类型
		docsUriDocs := make([]docsPkg.UriDoc, len(uriDocs))
		for i, doc := range uriDocs {
			docsUriDocs[i] = convertToDocsUriDoc(doc)
		}
		docsObjDocs := make([]docsPkg.WebObjectDoc, len(objDocs))
		for i, doc := range objDocs {
			docsObjDocs[i] = convertToDocsWebObjectDoc(doc)
		}
		spec := openapiGen.Generate(docsUriDocs, docsObjDocs)
		jsonData, err := spec.ToJSON()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.Data(http.StatusOK, "application/json", jsonData)
	})

	r.GET(prefix, func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(apiDocHTML))
	})
}

func GetDocDefine(obj any) *DocField {
	if obj == nil {
		return nil
	}
	rt := reflect.TypeOf(obj)
	f := parseDocField(rt, "", nil)
	return &f
}

func GetWebObjectDocDefine(prefix string, obj WebObject) WebObjectDoc {
	doc := WebObjectDoc{
		Group:        obj.Group,
		Path:         filepath.Join(prefix, obj.Name),
		Desc:         obj.Desc,
		AuthRequired: obj.AuthRequired,
		Filters:      obj.Filterables,
		Orders:       obj.Orderables,
		Searches:     obj.Searchables,
	}
	allowMethods := obj.AllowMethods
	if obj.AllowMethods == 0 {
		allowMethods = GET | CREATE | EDIT | DELETE | QUERY
	}

	if allowMethods&GET != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "GET")
	}
	if allowMethods&CREATE != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "CREATE")
	}
	if allowMethods&EDIT != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "EDIT")
	}
	if allowMethods&DELETE != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "DELETE")
	}
	if allowMethods&QUERY != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "QUERY")
	}

	doc.Fields = GetDocDefine(obj.Model).Fields
	allFields := []string{}
	for _, f := range doc.Fields {
		allFields = append(allFields, f.Name)
	}

	if len(obj.Editables) == 0 {
		doc.Editables = allFields
	} else {
		for _, ef := range obj.Editables {
			for _, f := range doc.Fields {
				if ef == f.FieldName {
					doc.Editables = append(doc.Editables, f.Name)
				}
			}
		}
	}

	for _, v := range obj.Views {
		doc.Views = append(doc.Views, UriDoc{
			Path:   filepath.Join(doc.Path, v.Path),
			Method: v.Method,
			Desc:   v.Desc,
		})
	}

	return doc
}

// parseDocField convert StructField Type to DocFiled.
func parseDocField(rt reflect.Type, name string, stacks []string) (val DocField) {
	val.Name = name
	val.Type = parseType(rt)

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		val.CanNull = true
	}

	if strings.HasPrefix(rt.Name(), "Null") {
		val.CanNull = true
	}

	if rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice {
		rt = rt.Elem()
		val.IsArray = true
	}

	switch rt.Name() {
	case "NullTime", "NullBool", "NullString", "NullByte", "NullInt16",
		"NullInt32", "NullInt64", "NullFloat32", "NullFloat64":
	case "Time", "DeletedAt":
		return val
	}

	if rt.Kind() != reflect.Struct {
		return val
	}

	val.Type = TYPE_OBJECT

	for _, v := range stacks {
		if rt.Name() == v {
			return val
		}
	}

	stacks = append(stacks, rt.Name())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i) // StructField
		jsonTag := f.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		// process embeded struct
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			embedRT := parseDocField(f.Type, "", stacks)
			val.Fields = append(val.Fields, embedRT.Fields...)
			continue
		}

		var name = f.Name
		if jsonTag != "" {
			name = strings.Split(jsonTag, ",")[0]
		}

		fieldRT := parseDocField(f.Type, name, stacks)
		fieldRT.FieldName = f.Name
		fieldRT.Desc = f.Tag.Get("comment")

		if strings.Contains(f.Tag.Get("binding"), "required") {
			fieldRT.Required = true
		}

		if strings.Contains(jsonTag, "omitempty") {
			fieldRT.CanNull = true
		}

		if strings.Contains(f.Tag.Get("gorm"), "primary") {
			fieldRT.IsPrimary = true
		}

		val.Fields = append(val.Fields, fieldRT)
	}
	return val
}

// parseType return type string according to reflect.Type.
func parseType(rt reflect.Type) string {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	// Multi-Level Pointers
	if rt.Kind() == reflect.Ptr {
		return TYPE_OBJECT
	}

	switch rt.Name() {
	case "NullTime", "Time", "DeletedAt":
		return TYPE_DATE
	}

	switch rt.Kind() {
	case reflect.Array, reflect.Slice:
		val := rt.Elem().Kind().String()
		if val == "struct" || val == "ptr" {
			val = TYPE_OBJECT
		}
		return val
	case reflect.String:
		return TYPE_STRING
	case reflect.Bool:
		return TYPE_BOOLEAN
	case reflect.Map:
		return TYPE_MAP
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return TYPE_INT
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return TYPE_FLOAT
	}

	return ""
}

// convertDocsUriDoc 转换 docs 包的 UriDoc 为 LingEcho 的 UriDoc
func convertDocsUriDoc(doc docsPkg.UriDoc) UriDoc {
	return UriDoc{
		Group:        doc.Group,
		Path:         doc.Path,
		Summary:      doc.Summary,
		Desc:         doc.Desc,
		AuthRequired: doc.AuthRequired,
		Method:       doc.Method,
		Request:      convertDocsDocField(doc.Request),
		Response:     convertDocsDocField(doc.Response),
	}
}

// convertToDocsUriDoc 转换 LingEcho 的 UriDoc 为 docs 包的 UriDoc
func convertToDocsUriDoc(doc UriDoc) docsPkg.UriDoc {
	return docsPkg.UriDoc{
		Group:        doc.Group,
		Path:         doc.Path,
		Summary:      doc.Summary,
		Desc:         doc.Desc,
		AuthRequired: doc.AuthRequired,
		Method:       doc.Method,
		Request:      convertToDocsDocField(doc.Request),
		Response:     convertToDocsDocField(doc.Response),
	}
}

// convertDocsDocField 转换 docs 包的 DocField 为 LingEcho 的 DocField
func convertDocsDocField(field *docsPkg.DocField) *DocField {
	if field == nil {
		return nil
	}
	result := &DocField{
		FieldName: field.FieldName,
		Name:      field.Name,
		Desc:      field.Desc,
		Type:      field.Type,
		Default:   field.Default,
		Required:  field.Required,
		CanNull:   field.CanNull,
		IsArray:   field.IsArray,
		IsPrimary: field.IsPrimary,
	}
	if len(field.Fields) > 0 {
		result.Fields = make([]DocField, len(field.Fields))
		for i, f := range field.Fields {
			result.Fields[i] = *convertDocsDocField(&f)
		}
	}
	return result
}

// convertToDocsDocField 转换 LingEcho 的 DocField 为 docs 包的 DocField
func convertToDocsDocField(field *DocField) *docsPkg.DocField {
	if field == nil {
		return nil
	}
	result := &docsPkg.DocField{
		FieldName: field.FieldName,
		Name:      field.Name,
		Desc:      field.Desc,
		Type:      field.Type,
		Default:   field.Default,
		Required:  field.Required,
		CanNull:   field.CanNull,
		IsArray:   field.IsArray,
		IsPrimary: field.IsPrimary,
	}
	if len(field.Fields) > 0 {
		result.Fields = make([]docsPkg.DocField, len(field.Fields))
		for i, f := range field.Fields {
			result.Fields[i] = *convertToDocsDocField(&f)
		}
	}
	return result
}

// convertToDocsWebObjectDoc 转换 LingEcho 的 WebObjectDoc 为 docs 包的 WebObjectDoc
func convertToDocsWebObjectDoc(doc WebObjectDoc) docsPkg.WebObjectDoc {
	result := docsPkg.WebObjectDoc{
		Group:        doc.Group,
		Path:         doc.Path,
		Desc:         doc.Desc,
		AuthRequired: doc.AuthRequired,
		AllowMethods: doc.AllowMethods,
		Filters:      doc.Filters,
		Orders:       doc.Orders,
		Searches:     doc.Searches,
		Editables:    doc.Editables,
	}
	if len(doc.Fields) > 0 {
		result.Fields = make([]docsPkg.DocField, len(doc.Fields))
		for i, f := range doc.Fields {
			result.Fields[i] = *convertToDocsDocField(&f)
		}
	}
	if len(doc.Views) > 0 {
		result.Views = make([]docsPkg.UriDoc, len(doc.Views))
		for i, v := range doc.Views {
			result.Views[i] = convertToDocsUriDoc(v)
		}
	}
	return result
}
