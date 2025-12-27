package handlers

import (
	"net/http"

	LingEcho "github.com/code-100-precent/LingFramework"
	"github.com/code-100-precent/LingFramework/internal/models"
	"github.com/code-100-precent/LingFramework/pkg/config"
	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/code-100-precent/LingFramework/pkg/search"
	"github.com/code-100-precent/LingFramework/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) GetObjs() []LingEcho.WebObject {
	return []LingEcho.WebObject{
		{
			Group:       "lingEcho",
			Desc:        "User",
			Model:       models.User{},
			Name:        "user",
			Filterables: []string{"UpdatedAt", "CreatedAt"},
			Editables:   []string{"Email", "Phone", "FirstName", "LastName", "DisplayName", "Role", "Permissions", "Enabled"},
			Searchables: []string{},
			Orderables:  []string{"UpdatedAt"},
			GetDB: func(c *gin.Context, isCreate bool) *gorm.DB {
				if isCreate {
					return h.db
				}
				return h.db.Where("deleted_at", nil)
			},
			BeforeCreate: func(db *gorm.DB, ctx *gin.Context, vptr any) error {
				return nil
			},
		},
	}
}

func (h *Handlers) GetDocs() []LingEcho.UriDoc {
	// Define the API documentation
	uriDocs := []LingEcho.UriDoc{
		{
			Group:        "User Authorization",
			Path:         config.GlobalConfig.APIPrefix + "/auth/logout",
			Method:       http.MethodGet,
			AuthRequired: true,
			Desc:         "User logout, if `?next={NEXT_URL}`is not empty, redirect to {NEXT_URL}",
		},
	}

	// 从数据库读取搜索配置，如果数据库中没有则使用配置文件
	searchEnabled := utils.GetBoolValue(h.db, constants.KEY_SEARCH_ENABLED)
	if !searchEnabled && config.GlobalConfig != nil {
		searchEnabled = config.GlobalConfig.SearchEnabled
	}

	if searchEnabled {
		uriDocs = append(uriDocs, []LingEcho.UriDoc{
			{
				Group:   "Search",
				Path:    config.GlobalConfig.APIPrefix + "/search",
				Method:  http.MethodPost,
				Desc:    "Execute a search query",
				Request: LingEcho.GetDocDefine(search.SearchRequest{}),
				Response: &LingEcho.DocField{
					Type: "object",
					Fields: []LingEcho.DocField{
						{Name: "Total", Type: LingEcho.TYPE_INT},
						{Name: "Took", Type: LingEcho.TYPE_INT},
						{Name: "Hits", Type: "array", Fields: []LingEcho.DocField{
							{Name: "ID", Type: LingEcho.TYPE_STRING},
							{Name: "Score", Type: LingEcho.TYPE_FLOAT},
							{Name: "Fields", Type: "object"},
						}},
					},
				},
			},
			{
				Group:        "Search",
				Path:         config.GlobalConfig.APIPrefix + "/search/index",
				Method:       http.MethodPost,
				AuthRequired: true,
				Desc:         "Index a new document",
				Request:      LingEcho.GetDocDefine(search.Doc{}),
				Response: &LingEcho.DocField{
					Type: LingEcho.TYPE_BOOLEAN,
					Desc: "true if document is indexed successfully",
				},
			},
			{
				Group:        "Search",
				Path:         config.GlobalConfig.APIPrefix + "/search/delete",
				Method:       http.MethodPost,
				AuthRequired: true,
				Desc:         "Delete a document by its ID",
				Request: &LingEcho.DocField{
					Type: "object",
					Fields: []LingEcho.DocField{
						{Name: "ID", Type: LingEcho.TYPE_STRING},
					},
				},
				Response: &LingEcho.DocField{
					Type: LingEcho.TYPE_BOOLEAN,
					Desc: "true if document is deleted successfully",
				},
			},
			{
				Group:        "Search",
				Path:         config.GlobalConfig.APIPrefix + "/search/auto-complete",
				Method:       http.MethodPost,
				AuthRequired: false,
				Desc:         "Get search query auto-completion suggestions",
				Request: &LingEcho.DocField{
					Type: "object",
					Fields: []LingEcho.DocField{
						{Name: "Keyword", Type: LingEcho.TYPE_STRING},
					},
				},
				Response: &LingEcho.DocField{
					Type: "object",
					Fields: []LingEcho.DocField{
						{Name: "suggestions", Type: "array", Fields: []LingEcho.DocField{
							{Name: "suggestion", Type: LingEcho.TYPE_STRING},
						}},
					},
				},
			},
			{
				Group:        "Search",
				Path:         config.GlobalConfig.APIPrefix + "/search/suggest",
				Method:       http.MethodPost,
				AuthRequired: false,
				Desc:         "Get search suggestions based on the keyword",
				Request: &LingEcho.DocField{
					Type: "object",
					Fields: []LingEcho.DocField{
						{Name: "Keyword", Type: LingEcho.TYPE_STRING},
					},
				},
				Response: &LingEcho.DocField{
					Type: "object",
					Fields: []LingEcho.DocField{
						{Name: "suggestions", Type: "array", Fields: []LingEcho.DocField{
							{Name: "suggestion", Type: LingEcho.TYPE_STRING},
						}},
					},
				},
			},
		}...)
	}
	return uriDocs
}
