package constants

const (
	//SigUserLogin: user *User, c *gin.Context
	SigUserLogin = "user.login"
	//SigUserLogout: user *User, c *gin.Context
	SigUserLogout = "user.logout"
	//SigUserCreate: user *User, c *gin.Context
	SigUserCreate = "user.create"
	//SigUserVerifyEmail: user *User, hash, clientIp, userAgent string
	SigUserVerifyEmail = "user.verifyemail"
	//SigUserResetPassword: user *User, hash, clientIp, userAgent string
	SigUserResetPassword = "user.resetpassword"
	//SigUserChangeEmail: user *User, hash, clientIp, userAgent, newEmail string
	SigUserChangeEmail = "user.changeemail"
	//SigUserChangeEmailDone: user *User, oldEmail, newEmail string
	SigUserChangeEmailDone = "user.changeemaildone"
)
