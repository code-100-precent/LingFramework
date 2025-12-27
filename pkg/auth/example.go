package auth

// This file contains usage examples for the auth package.
// These are commented code examples for reference.

/*
Example 1: Basic setup with JWT and RBAC

	// Load configuration from environment variables
	config, err := LoadAuthConfigFromEnv()
	if err != nil {
		log.Fatal("Failed to load auth config:", err)
	}

	// Create auth manager
	authManager, err := NewAuthManager(config)
	if err != nil {
		log.Fatal("Failed to create auth manager:", err)
	}

	// Setup middleware config
	middlewareConfig := DefaultMiddlewareConfig(authManager)

	// Use in Gin router
	router := gin.Default()
	router.Use(AuthMiddleware(middlewareConfig))

	// Protected route
	router.GET("/api/users", func(c *gin.Context) {
		userID, _ := GetUserID(c)
		username, _ := GetUsername(c)
		c.JSON(200, gin.H{"user_id": userID, "username": username})
	})

	// Route with permission check
	router.DELETE("/api/users/:id",
		PermissionMiddleware(middlewareConfig, "user", "delete"),
		func(c *gin.Context) {
			// Delete user logic
		},
	)


Example 2: Setup RBAC permissions

	// Get RBAC manager
	rbac := authManager.GetRBAC()

	// Define permissions
	adminPerms := []Permission{
		{Resource: "user", Action: "read"},
		{Resource: "user", Action: "write"},
		{Resource: "user", Action: "delete"},
		{Resource: "article", Action: "read"},
		{Resource: "article", Action: "write"},
		{Resource: "article", Action: "delete"},
	}

	userPerms := []Permission{
		{Resource: "article", Action: "read"},
		{Resource: "article", Action: "write"},
	}

	// Add roles
	rbac.AddRole("admin", adminPerms)
	rbac.AddRole("user", userPerms)

	// Assign roles to users
	rbac.AssignRole(1, "admin")  // User ID 1 is admin
	rbac.AssignRole(2, "user")   // User ID 2 is regular user


Example 3: Setup ABAC policies

	// Get ABAC manager
	abac := authManager.GetABAC()

	// Create a policy that allows users to edit their own articles
	policy := &Policy{
		ID:          "edit-own-article",
		Name:        "Edit Own Article",
		Description: "Users can edit articles they own",
		Subjects: []Attribute{
			{Key: "id", Value: "*"}, // Any user
		},
		Resources: []Attribute{
			{Key: "type", Value: "article"},
			{Key: "owner_id", Value: "*"}, // Will be matched with subject.id
		},
		Actions:    []string{"edit"},
		Effect:     "allow",
		Conditions: []Condition{
			{
				Attribute: "subject.id",
				Operator:  "eq",
				Value:     "{{resource.owner_id}}", // Match user ID with article owner
			},
		},
		Priority: 100,
	}

	abac.AddPolicy(policy)


Example 4: Generate JWT tokens

	// Generate access and refresh tokens
	accessToken, refreshToken, err := authManager.GenerateToken(
		1,                    // userID
		"john_doe",           // username
		[]string{"admin"},    // roles
		nil,                  // extra claims
	)
	if err != nil {
		log.Fatal("Failed to generate tokens:", err)
	}


Example 5: Refresh token

	// Refresh access token using refresh token
	newAccessToken, err := authManager.RefreshToken(refreshToken)
	if err != nil {
		log.Fatal("Failed to refresh token:", err)
	}


Example 6: OAuth2 Server Setup

	// When using OAuth2 as auth type
	oauth2Server := authManager.GetOAuth2Server()

	// Register OAuth2 client
	oauth2Server.RegisterClient(
		"client-id",
		"client-secret",
		"https://example.com/callback",
		[]string{"read", "write"},
	)

	// Generate authorization code
	code, err := oauth2Server.GenerateAuthorizationCode(
		"client-id",
		1, // userID
		"https://example.com/callback",
		[]string{"read", "write"},
	)

	// Exchange code for tokens
	tokenInfo, err := oauth2Server.ExchangeCode(code, "client-id", "client-secret")


Example 7: Custom resource attributes for ABAC

	router.DELETE("/api/articles/:id",
		PermissionMiddlewareWithAttrs(
			middlewareConfig,
			"article",
			"delete",
			func(c *gin.Context) map[string]interface{} {
				// Load article from database
				articleID := c.Param("id")
				article := getArticleFromDB(articleID)

				return map[string]interface{}{
					"type":     "article",
					"id":       articleID,
					"owner_id": article.OwnerID,
					"status":   article.Status,
				}
			},
		),
		func(c *gin.Context) {
			// Delete article logic
		},
	)


Environment Variables:

Required (when using JWT):
- JWT_SECRET_KEY: Secret key for signing JWT tokens (or use SESSION_SECRET as fallback)

Optional:
- AUTH_TYPE: Authentication type - "jwt" or "oauth2" (default: "jwt")
- PERMISSION_TYPE: Permission type - "rbac" or "abac" (default: "rbac")
- AUTH_TOKEN_HEADER: Token header name (default: "Authorization")
- AUTH_TOKEN_PREFIX: Token prefix (default: "Bearer ")
- AUTH_SKIP_PATHS: Additional skip paths, comma-separated
- JWT_ACCESS_TOKEN_TTL: Access token TTL in seconds (default: 900)
- JWT_REFRESH_TOKEN_TTL: Refresh token TTL in seconds (default: 604800)
- JWT_ISSUER: JWT issuer (default: "LingFramework")
- OAUTH2_ACCESS_TOKEN_TTL: OAuth2 access token TTL in seconds (default: 900)
- OAUTH2_REFRESH_TOKEN_TTL: OAuth2 refresh token TTL in seconds (default: 604800)
*/

// This file only contains documentation/comments, no actual code
