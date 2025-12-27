package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewOAuth2Server(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	assert.NotNil(t, server)
	assert.Equal(t, 15*time.Minute, server.accessTokenTTL)
	assert.Equal(t, 7*24*time.Hour, server.refreshTokenTTL)
}

func TestOAuth2Server_RegisterClient(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)

	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read", "write"})

	client, exists := server.GetClient("client-1")
	assert.True(t, exists)
	assert.NotNil(t, client)
	assert.Equal(t, "client-1", client.ID)
	assert.Equal(t, "secret-1", client.Secret)
	assert.Equal(t, "http://example.com/callback", client.RedirectURI)
	assert.Equal(t, []string{"read", "write"}, client.Scopes)
}

func TestOAuth2Server_GetClient_NonExistent(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)

	client, exists := server.GetClient("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, client)
}

func TestOAuth2Server_GenerateAuthorizationCode(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)
	assert.NotEmpty(t, code)
}

func TestOAuth2Server_GenerateAuthorizationCode_InvalidClient(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)

	_, err := server.GenerateAuthorizationCode("invalid-client", 1, "http://example.com/callback", []string{"read"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestOAuth2Server_GenerateAuthorizationCode_RedirectURIMismatch(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	_, err := server.GenerateAuthorizationCode("client-1", 1, "http://wrong.com/callback", []string{"read"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redirect URI mismatch")
}

func TestOAuth2Server_ExchangeCode(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	tokenInfo, err := server.ExchangeCode(code, "client-1", "secret-1")
	assert.NoError(t, err)
	assert.NotNil(t, tokenInfo)
	assert.NotEmpty(t, tokenInfo.AccessToken)
	assert.NotEmpty(t, tokenInfo.RefreshToken)
	assert.Equal(t, uint(1), tokenInfo.UserID)
	assert.Equal(t, "client-1", tokenInfo.ClientID)
}

func TestOAuth2Server_ExchangeCode_InvalidCode(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	_, err := server.ExchangeCode("invalid-code", "client-1", "secret-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid authorization code")
}

func TestOAuth2Server_ExchangeCode_InvalidClient(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	_, err = server.ExchangeCode(code, "wrong-client", "secret-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client credentials")
}

func TestOAuth2Server_ExchangeCode_WrongSecret(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	_, err = server.ExchangeCode(code, "client-1", "wrong-secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client credentials")
}

func TestOAuth2Server_ValidateToken(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	tokenInfo, err := server.ExchangeCode(code, "client-1", "secret-1")
	assert.NoError(t, err)

	validated, err := server.ValidateToken(tokenInfo.AccessToken)
	assert.NoError(t, err)
	assert.NotNil(t, validated)
	assert.Equal(t, tokenInfo.AccessToken, validated.AccessToken)
}

func TestOAuth2Server_ValidateToken_Invalid(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)

	_, err := server.ValidateToken("invalid-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestOAuth2Server_RefreshAccessToken(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	tokenInfo, err := server.ExchangeCode(code, "client-1", "secret-1")
	assert.NoError(t, err)

	oldAccessToken := tokenInfo.AccessToken
	newTokenInfo, err := server.RefreshAccessToken(tokenInfo.RefreshToken)
	assert.NoError(t, err)
	assert.NotNil(t, newTokenInfo)
	assert.NotEqual(t, oldAccessToken, newTokenInfo.AccessToken)
	assert.Equal(t, tokenInfo.RefreshToken, newTokenInfo.RefreshToken)

	// Old token should be invalid
	_, err = server.ValidateToken(oldAccessToken)
	assert.Error(t, err)

	// New token should be valid
	_, err = server.ValidateToken(newTokenInfo.AccessToken)
	assert.NoError(t, err)
}

func TestOAuth2Server_RefreshAccessToken_Invalid(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)

	_, err := server.RefreshAccessToken("invalid-refresh-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestOAuth2Server_RevokeToken(t *testing.T) {
	server := NewOAuth2Server(15*time.Minute, 7*24*time.Hour)
	server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	tokenInfo, err := server.ExchangeCode(code, "client-1", "secret-1")
	assert.NoError(t, err)

	server.RevokeToken(tokenInfo.AccessToken)

	_, err = server.ValidateToken(tokenInfo.AccessToken)
	assert.Error(t, err)

	_, err = server.RefreshAccessToken(tokenInfo.RefreshToken)
	assert.Error(t, err)
}

func TestNewOAuth2Client(t *testing.T) {
	config := &OAuth2ServerConfig{
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		RedirectURL:  "http://example.com/callback",
		Scopes:       []string{"read", "write"},
		AuthURL:      "http://auth.example.com/authorize",
		TokenURL:     "http://auth.example.com/token",
	}

	client := NewOAuth2Client(config, "test-provider")
	assert.NotNil(t, client)
	assert.NotNil(t, client.provider)
	assert.Equal(t, "test-provider", client.provider.Name)
}

func TestOAuth2Client_GetAuthURL(t *testing.T) {
	config := &OAuth2ServerConfig{
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		RedirectURL:  "http://example.com/callback",
		Scopes:       []string{"read"},
		AuthURL:      "http://auth.example.com/authorize",
		TokenURL:     "http://auth.example.com/token",
	}

	client := NewOAuth2Client(config, "test-provider")
	authURL := client.GetAuthURL("test-state")

	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "http://auth.example.com/authorize")
	assert.Contains(t, authURL, "test-state")
}

func TestGenerateRandomString(t *testing.T) {
	str, err := generateRandomString(32)
	assert.NoError(t, err)
	assert.Equal(t, 32, len(str))

	// Generate another one to ensure randomness
	str2, err := generateRandomString(32)
	assert.NoError(t, err)
	assert.NotEqual(t, str, str2)
}
