package handlers

import (
	"net/http"
	"time"

	LingEcho "github.com/code-100-precent/LingFramework"
	"github.com/code-100-precent/LingFramework/internal/models"
	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/code-100-precent/LingFramework/pkg/utils"
	"github.com/code-100-precent/LingFramework/pkg/utils/response"
	"github.com/gin-gonic/gin"
)

// handleUserSignupPage handle user signup page
func (h *Handlers) handleUserSignupPage(c *gin.Context) {
	ctx := LingEcho.GetRenderPageContext(c)
	ctx["SignupText"] = "Sign Up Now"
	ctx["Site.SignupApi"] = utils.GetValue(h.db, constants.KEY_SITE_SIGNUP_API)
	c.HTML(http.StatusOK, "signup.html", ctx)
}

// handleUserResetPasswordPage handle user reset password page
func (h *Handlers) handleUserResetPasswordPage(c *gin.Context) {
	c.HTML(http.StatusOK, "reset_password.html", LingEcho.GetRenderPageContext(c))
}

// handleUserSigninPage handle user signin page
func (h *Handlers) handleUserSigninPage(c *gin.Context) {
	ctx := LingEcho.GetRenderPageContext(c)
	ctx["SignupText"] = "Sign Up Now"
	c.HTML(http.StatusOK, "signin.html", ctx)
}

// handleUserLogout handle user logout
func (h *Handlers) handleUserLogout(c *gin.Context) {
	user := models.CurrentUser(c)
	if user != nil {
		models.Logout(c, user)
	}
	next := c.Query("next")
	if next != "" {
		c.Redirect(http.StatusFound, next)
		return
	}
	response.Success(c, "Logout Success", nil)
}

// handleUserInfo handle user info
func (h *Handlers) handleUserInfo(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	withToken := c.Query("with_token")
	if withToken != "" {
		expired, err := time.ParseDuration(withToken)
		if err == nil {
			if expired >= 24*time.Hour {
				expired = 24 * time.Hour
			}
			user.AuthToken = ""
		}
	}
	response.Success(c, "success", user)
}
