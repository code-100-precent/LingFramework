package bootstrap

import (
	"strconv"

	"github.com/code-100-precent/LingFramework/pkg/config"
	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/code-100-precent/LingFramework/pkg/utils"
	"gorm.io/gorm"
)

type SeedService struct {
	db *gorm.DB
}

func (s *SeedService) SeedAll() error {
	if err := s.seedConfigs(); err != nil {
		return err
	}

	return nil
}

func (s *SeedService) seedConfigs() error {
	apiPrefix := config.GlobalConfig.APIPrefix
	defaults := []utils.Config{
		{Key: constants.KEY_SITE_URL, Desc: "Site URL", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerUrl != "" {
				return config.GlobalConfig.ServerUrl
			}
			return "https://lingecho.com"
		}()},
		{Key: constants.KEY_SITE_NAME, Desc: "Site Name", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerName != "" {
				return config.GlobalConfig.ServerName
			}
			return "LingEcho"
		}()},
		{Key: constants.KEY_SITE_LOGO_URL, Desc: "Site Logo", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerLogo != "" {
				return config.GlobalConfig.ServerLogo
			}
			return "/static/img/favicon.png"
		}()},
		{Key: constants.KEY_SITE_DESCRIPTION, Desc: "Site Description", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerDesc != "" {
				return config.GlobalConfig.ServerDesc
			}
			return "LingEcho - Intelligent Voice Customer Service Platform"
		}()},
		{Key: constants.KEY_SITE_TERMS_URL, Desc: "Terms of Service", Autoload: true, Public: true, Format: "text", Value: func() string {
			if config.GlobalConfig.ServerTermsUrl != "" {
				return config.GlobalConfig.ServerTermsUrl
			}
			return "https://lingecho.com"
		}()},
		{Key: constants.KEY_SITE_SIGNIN_URL, Desc: "Sign In Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/login"},
		{Key: constants.KEY_SITE_FAVICON_URL, Desc: "Favicon URL", Autoload: true, Public: true, Format: "text", Value: "/static/img/favicon.png"},
		{Key: constants.KEY_SITE_SIGNUP_URL, Desc: "Sign Up Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/register"},
		{Key: constants.KEY_SITE_LOGOUT_URL, Desc: "Logout Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/logout"},
		{Key: constants.KEY_SITE_RESET_PASSWORD_URL, Desc: "Reset Password Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/reset-password"},
		{Key: constants.KEY_SITE_SIGNIN_API, Desc: "Sign In API", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/login"},
		{Key: constants.KEY_SITE_SIGNUP_API, Desc: "Sign Up API", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/register"},
		{Key: constants.KEY_SITE_RESET_PASSWORD_DONE_API, Desc: "Reset Password API", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/auth/reset-password-done"},
		{Key: constants.KEY_SITE_LOGIN_NEXT, Desc: "Login Redirect Page", Autoload: true, Public: true, Format: "text", Value: apiPrefix + "/admin/"},
		{Key: constants.KEY_SITE_USER_ID_TYPE, Desc: "User ID Type", Autoload: true, Public: true, Format: "text", Value: "email"},
		// Search configuration
		{Key: constants.KEY_SEARCH_ENABLED, Desc: "Search Feature Enabled", Autoload: true, Public: true, Format: "bool", Value: func() string {
			if config.GlobalConfig.SearchEnabled {
				return "true"
			}
			return "false"
		}()},
		{Key: constants.KEY_SEARCH_PATH, Desc: "Search Index Path", Autoload: true, Public: false, Format: "text", Value: func() string {
			if config.GlobalConfig.SearchPath != "" {
				return config.GlobalConfig.SearchPath
			}
			return "./search"
		}()},
		{Key: constants.KEY_SEARCH_BATCH_SIZE, Desc: "Search Batch Size", Autoload: true, Public: false, Format: "int", Value: func() string {
			if config.GlobalConfig.SearchBatchSize > 0 {
				return strconv.Itoa(config.GlobalConfig.SearchBatchSize)
			}
			return "100"
		}()},
		{Key: constants.KEY_SEARCH_INDEX_SCHEDULE, Desc: "Search Index Schedule (Cron)", Autoload: true, Public: false, Format: "text", Value: "0 */6 * * *"}, // Execute every 6 hours
	}
	for _, cfg := range defaults {
		var count int64
		err := s.db.Model(&utils.Config{}).Where("`key` = ?", cfg.Key).Count(&count).Error
		if err != nil {
			return err
		}
		if count == 0 {
			if err := s.db.Create(&cfg).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
