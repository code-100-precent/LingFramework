package models

import (
	"time"

	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/code-100-precent/LingFramework/pkg/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type User struct {
	BaseModel
	Email                 string     `json:"email" gorm:"size:128;uniqueIndex"`
	Password              string     `json:"-" gorm:"size:128"`
	Phone                 string     `json:"phone,omitempty" gorm:"size:64;index"`
	FirstName             string     `json:"firstName,omitempty" gorm:"size:128"`
	LastName              string     `json:"lastName,omitempty" gorm:"size:128"`
	DisplayName           string     `json:"displayName,omitempty" gorm:"size:128"`
	IsStaff               bool       `json:"isStaff,omitempty"`
	Enabled               bool       `json:"-"`
	Activated             bool       `json:"-"`
	LastLogin             *time.Time `json:"lastLogin,omitempty"`
	LastLoginIP           string     `json:"-" gorm:"size:128"`
	Source                string     `json:"-" gorm:"size:64;index"`
	Locale                string     `json:"locale,omitempty" gorm:"size:20"`
	Timezone              string     `json:"timezone,omitempty" gorm:"size:200"`
	AuthToken             string     `json:"token,omitempty" gorm:"-"`
	Avatar                string     `json:"avatar,omitempty"`
	Gender                string     `json:"gender,omitempty"`
	City                  string     `json:"city,omitempty"`
	Region                string     `json:"region,omitempty"`
	Country               string     `json:"country,omitempty"`
	HasFilledDetails      bool       `json:"hasFilledDetails"`
	EmailNotifications    bool       `json:"emailNotifications"`                           // 邮件通知
	PushNotifications     bool       `json:"pushNotifications" gorm:"default:true"`        // 推送通知
	SystemNotifications   bool       `json:"systemNotifications" gorm:"default:true"`      // 系统通知
	AutoCleanUnreadEmails bool       `json:"autoCleanUnreadEmails" gorm:"default:false"`   // 自动清理七天未读邮件
	EmailVerified         bool       `json:"emailVerified" gorm:"default:false"`           // 邮箱已验证
	PhoneVerified         bool       `json:"phoneVerified" gorm:"default:false"`           // 手机已验证
	TwoFactorEnabled      bool       `json:"twoFactorEnabled" gorm:"default:false"`        // 双因素认证
	TwoFactorSecret       string     `json:"-" gorm:"size:128"`                            // 双因素认证密钥
	EmailVerifyToken      string     `json:"-" gorm:"size:128"`                            // 邮箱验证令牌
	PhoneVerifyToken      string     `json:"-" gorm:"size:128"`                            // 手机验证令牌
	PasswordResetToken    string     `json:"-" gorm:"size:128"`                            // 密码重置令牌
	PasswordResetExpires  *time.Time `json:"-"`                                            // 密码重置过期时间
	EmailVerifyExpires    *time.Time `json:"-"`                                            // 邮箱验证过期时间
	LoginCount            int        `json:"loginCount" gorm:"default:0"`                  // 登录次数
	LastPasswordChange    *time.Time `json:"lastPasswordChange,omitempty"`                 // 最后密码修改时间
	ProfileComplete       int        `json:"profileComplete" gorm:"default:0"`             // 资料完整度百分比
	Role                  string     `json:"role,omitempty" gorm:"size:50;default:'user'"` // 用户角色
	Permissions           string     `json:"permissions,omitempty" gorm:"type:text"`       // 用户权限JSON
}

func CurrentUser(c *gin.Context) *User {
	if cachedObj, exists := c.Get(constants.UserField); exists && cachedObj != nil {
		return cachedObj.(*User)
	}
	session := sessions.Default(c)
	userId := session.Get(constants.UserField)
	if userId == nil {
		return nil
	}
	db := c.MustGet(constants.DbField).(*gorm.DB)
	user, err := GetUserByUID(db, userId.(uint))
	if err != nil {
		return nil
	}
	c.Set(constants.UserField, user)
	return user
}

func GetUserByUID(db *gorm.DB, userID uint) (*User, error) {
	var val User
	result := db.Where("id", userID).Where("enabled", true).Take(&val)
	if result.Error != nil {
		return nil, result.Error
	}
	return &val, nil
}

func Logout(c *gin.Context, user *User) {
	c.Set(constants.UserField, nil)
	session := sessions.Default(c)
	session.Delete(constants.UserField)
	session.Save()
	utils.Sig().Emit(constants.SigUserLogout, user, c)
}
