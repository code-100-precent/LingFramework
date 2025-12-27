package handlers

import (
	"log"
	"time"

	LingEcho "github.com/code-100-precent/LingFramework"
	"github.com/code-100-precent/LingFramework/pkg/config"
	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/code-100-precent/LingFramework/pkg/logger"
	"github.com/code-100-precent/LingFramework/pkg/middleware"
	"github.com/code-100-precent/LingFramework/pkg/search"
	"github.com/code-100-precent/LingFramework/pkg/utils"
	"github.com/code-100-precent/LingFramework/pkg/websocket"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handlers struct {
	db            *gorm.DB
	wsHub         *websocket.Hub
	searchHandler *search.SearchHandlers
}

// GetSearchHandler gets the search handlers (for scheduled tasks)
func (h *Handlers) GetSearchHandler() *search.SearchHandlers {
	return h.searchHandler
}

func NewHandlers(db *gorm.DB) *Handlers {
	wsConfig := websocket.LoadConfigFromEnv()
	wsHub := websocket.NewHub(wsConfig)
	var searchHandler *search.SearchHandlers

	// Read search configuration from config table
	searchEnabled := utils.GetBoolValue(db, constants.KEY_SEARCH_ENABLED)
	// If not configured in config table, use environment variables
	if !searchEnabled && config.GlobalConfig != nil {
		searchEnabled = config.GlobalConfig.SearchEnabled
	}

	if searchEnabled {
		searchPath := utils.GetValue(db, constants.KEY_SEARCH_PATH)
		if searchPath == "" && config.GlobalConfig != nil {
			searchPath = config.GlobalConfig.SearchPath
		}
		if searchPath == "" {
			searchPath = "./search"
		}

		batchSize := utils.GetIntValue(db, constants.KEY_SEARCH_BATCH_SIZE, 100)
		if batchSize == 0 && config.GlobalConfig != nil {
			batchSize = config.GlobalConfig.SearchBatchSize
		}
		if batchSize == 0 {
			batchSize = 100
		}

		engine, err := search.New(
			search.Config{
				IndexPath:    searchPath,
				QueryTimeout: 5 * time.Second,
				BatchSize:    batchSize,
			},
			search.BuildIndexMapping(""),
		)
		if err != nil {
			log.Printf("Failed to initialize search engine: %v", err)
			// Even if initialization fails, create an empty handlers for route registration
			searchHandler = search.NewSearchHandlers(nil)
		} else {
			searchHandler = search.NewSearchHandlers(engine)
		}
		// Set database connection for configuration checking
		if searchHandler != nil {
			searchHandler.SetDB(db)
		}
	} else {
		// Even if search is not enabled, create an empty handlers for route registration
		searchHandler = search.NewSearchHandlers(nil)
		if searchHandler != nil {
			searchHandler.SetDB(db)
		}
	}

	return &Handlers{
		db:            db,
		wsHub:         wsHub,
		searchHandler: searchHandler,
	}
}

func (h *Handlers) Register(engine *gin.Engine) {

	r := engine.Group(config.GlobalConfig.APIPrefix)

	// Register Global Singleton DB
	r.Use(middleware.InjectDB(h.db))

	// Register Operation Log Middleware for authenticated routes
	r.Use(middleware.OperationLogMiddleware())

	// Register routes regardless of whether search is enabled, check in handlers methods
	// If handlers is nil, try to initialize
	if h.searchHandler == nil {
		searchPath := utils.GetValue(h.db, constants.KEY_SEARCH_PATH)
		if searchPath == "" && config.GlobalConfig != nil {
			searchPath = config.GlobalConfig.SearchPath
		}
		if searchPath == "" {
			searchPath = "./search"
		}

		batchSize := utils.GetIntValue(h.db, constants.KEY_SEARCH_BATCH_SIZE, 100)
		if batchSize == 0 && config.GlobalConfig != nil {
			batchSize = config.GlobalConfig.SearchBatchSize
		}
		if batchSize == 0 {
			batchSize = 100
		}

		engine, err := search.New(
			search.Config{
				IndexPath:    searchPath,
				QueryTimeout: 5 * time.Second,
				BatchSize:    batchSize,
			},
			search.BuildIndexMapping(""),
		)
		if err != nil {
			logger.Warn("Failed to initialize search engine in Register", zap.Error(err))
			// Even if initialization fails, create an empty handlers for route registration
			h.searchHandler = search.NewSearchHandlers(nil)
		} else {
			h.searchHandler = search.NewSearchHandlers(engine)
		}
	}

	// Register routes regardless of whether search is enabled, check in handlers methods
	if h.searchHandler == nil {
		// If handlers is still nil, create an empty one for route registration
		logger.Info("Search handlers is nil, creating empty handlers for route registration")
		h.searchHandler = search.NewSearchHandlers(nil)
	}

	// Set database connection for configuration checking
	if h.searchHandler != nil {
		h.searchHandler.SetDB(h.db)
		logger.Info("Registering search routes")
		h.searchHandler.RegisterSearchRoutes(r)
		logger.Info("Search routes registered successfully")
	} else {
		logger.Warn("Search handlers is still nil after initialization, routes not registered")
	}
	objs := h.GetObjs()
	LingEcho.RegisterObjects(r, objs)
	if config.GlobalConfig.DocsPrefix != "" {
		var objDocs []LingEcho.WebObjectDoc
		for _, obj := range objs {
			objDocs = append(objDocs, LingEcho.GetWebObjectDocDefine(config.GlobalConfig.APIPrefix, obj))
		}
		LingEcho.RegisterHandler(config.GlobalConfig.DocsPrefix, engine, h.GetDocs(), objDocs, h.db)
	}
}
