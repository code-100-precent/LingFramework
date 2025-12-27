package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// OAuth2Provider represents an OAuth2 provider
type OAuth2Provider struct {
	Config *oauth2.Config
	Name   string
}

// OAuth2ServerConfig represents OAuth2 server configuration
type OAuth2ServerConfig struct {
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	Scopes          []string
	AuthURL         string
	TokenURL        string
	UserInfoURL     string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// OAuth2Client represents an OAuth2 client
type OAuth2Client struct {
	provider *OAuth2Provider
}

// NewOAuth2Client creates a new OAuth2 client
func NewOAuth2Client(config *OAuth2ServerConfig, providerName string) *OAuth2Client {
	provider := &OAuth2Provider{
		Name: providerName,
		Config: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       config.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  config.AuthURL,
				TokenURL: config.TokenURL,
			},
		},
	}

	return &OAuth2Client{provider: provider}
}

// GetAuthURL returns the authorization URL
func (c *OAuth2Client) GetAuthURL(state string) string {
	return c.provider.Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges authorization code for token
func (c *OAuth2Client) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.provider.Config.Exchange(ctx, code)
}

// GetTokenSource returns a token source for the given token
func (c *OAuth2Client) GetTokenSource(ctx context.Context, token *oauth2.Token) oauth2.TokenSource {
	return c.provider.Config.TokenSource(ctx, token)
}

// RefreshToken refreshes an access token
func (c *OAuth2Client) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	return c.provider.Config.TokenSource(ctx, token).Token()
}

// OAuth2Server represents an OAuth2 authorization server
type OAuth2Server struct {
	mu              sync.RWMutex
	clients         map[string]*ClientInfo        // clientID -> client info
	codes           map[string]*AuthorizationCode // code -> authorization code
	tokens          map[string]*TokenInfo         // accessToken -> token info
	refreshTokens   map[string]*TokenInfo         // refreshToken -> token info
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// ClientInfo represents OAuth2 client information
type ClientInfo struct {
	ID          string
	Secret      string
	RedirectURI string
	Scopes      []string
}

// AuthorizationCode represents an authorization code
type AuthorizationCode struct {
	Code        string
	ClientID    string
	UserID      uint
	RedirectURI string
	Scopes      []string
	ExpiresAt   time.Time
}

// TokenInfo represents token information
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	ClientID     string
	UserID       uint
	Scopes       []string
	ExpiresAt    time.Time
}

// NewOAuth2Server creates a new OAuth2 server
func NewOAuth2Server(accessTokenTTL, refreshTokenTTL time.Duration) *OAuth2Server {
	return &OAuth2Server{
		clients:         make(map[string]*ClientInfo),
		codes:           make(map[string]*AuthorizationCode),
		tokens:          make(map[string]*TokenInfo),
		refreshTokens:   make(map[string]*TokenInfo),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// RegisterClient registers a new OAuth2 client
func (s *OAuth2Server) RegisterClient(clientID, clientSecret, redirectURI string, scopes []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[clientID] = &ClientInfo{
		ID:          clientID,
		Secret:      clientSecret,
		RedirectURI: redirectURI,
		Scopes:      scopes,
	}
}

// GetClient retrieves client information
func (s *OAuth2Server) GetClient(clientID string) (*ClientInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[clientID]
	return client, ok
}

// GenerateAuthorizationCode generates an authorization code
func (s *OAuth2Server) GenerateAuthorizationCode(clientID string, userID uint, redirectURI string, scopes []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate client
	client, ok := s.clients[clientID]
	if !ok {
		return "", errors.New("invalid client")
	}

	if redirectURI != client.RedirectURI {
		return "", errors.New("redirect URI mismatch")
	}

	// Generate random code
	code, err := generateRandomString(32)
	if err != nil {
		return "", err
	}

	authCode := &AuthorizationCode{
		Code:        code,
		ClientID:    clientID,
		UserID:      userID,
		RedirectURI: redirectURI,
		Scopes:      scopes,
		ExpiresAt:   time.Now().Add(10 * time.Minute), // Authorization code expires in 10 minutes
	}

	s.codes[code] = authCode

	// Clean up expired codes
	go s.cleanupExpiredCodes()

	return code, nil
}

// ExchangeCode exchanges authorization code for access token
func (s *OAuth2Server) ExchangeCode(code, clientID, clientSecret string) (*TokenInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate client
	client, ok := s.clients[clientID]
	if !ok || client.Secret != clientSecret {
		return nil, errors.New("invalid client credentials")
	}

	// Find and validate code
	authCode, ok := s.codes[code]
	if !ok {
		return nil, errors.New("invalid authorization code")
	}

	if authCode.ClientID != clientID {
		return nil, errors.New("client ID mismatch")
	}

	if time.Now().After(authCode.ExpiresAt) {
		delete(s.codes, code)
		return nil, errors.New("authorization code expired")
	}

	// Generate tokens
	accessToken, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	tokenInfo := &TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ClientID:     clientID,
		UserID:       authCode.UserID,
		Scopes:       authCode.Scopes,
		ExpiresAt:    time.Now().Add(s.accessTokenTTL),
	}

	// Store tokens
	s.tokens[accessToken] = tokenInfo
	s.refreshTokens[refreshToken] = tokenInfo

	// Delete used code
	delete(s.codes, code)

	return tokenInfo, nil
}

// ValidateToken validates an access token
func (s *OAuth2Server) ValidateToken(accessToken string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokenInfo, ok := s.tokens[accessToken]
	if !ok {
		return nil, errors.New("invalid token")
	}

	if time.Now().After(tokenInfo.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	return tokenInfo, nil
}

// RefreshAccessToken refreshes an access token using refresh token
func (s *OAuth2Server) RefreshAccessToken(refreshToken string) (*TokenInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenInfo, ok := s.refreshTokens[refreshToken]
	if !ok {
		return nil, errors.New("invalid refresh token")
	}

	// Generate new access token
	newAccessToken, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	// Delete old access token
	delete(s.tokens, tokenInfo.AccessToken)

	// Update token info
	tokenInfo.AccessToken = newAccessToken
	tokenInfo.ExpiresAt = time.Now().Add(s.accessTokenTTL)

	// Store new token
	s.tokens[newAccessToken] = tokenInfo

	return tokenInfo, nil
}

// RevokeToken revokes an access token
func (s *OAuth2Server) RevokeToken(accessToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tokenInfo, ok := s.tokens[accessToken]; ok {
		delete(s.tokens, accessToken)
		delete(s.refreshTokens, tokenInfo.RefreshToken)
	}
}

// cleanupExpiredCodes removes expired authorization codes
func (s *OAuth2Server) cleanupExpiredCodes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for code, authCode := range s.codes {
		if now.After(authCode.ExpiresAt) {
			delete(s.codes, code)
		}
	}
}

// generateRandomString generates a random string
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// ClientCredentialsConfig represents client credentials configuration
type ClientCredentialsConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// GetClientCredentialsToken gets a token using client credentials flow
func GetClientCredentialsToken(ctx context.Context, config *ClientCredentialsConfig) (*oauth2.Token, error) {
	ccConfig := &clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.TokenURL,
		Scopes:       config.Scopes,
	}

	return ccConfig.Token(ctx)
}

// OAuth2Middleware creates OAuth2 authentication middleware
func OAuth2Middleware(server *OAuth2Server) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
				http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
				return
			}

			token := authHeader[7:]
			tokenInfo, err := server.ValidateToken(token)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
				return
			}

			// Add token info to request context
			ctx := context.WithValue(r.Context(), "oauth2_token", tokenInfo)
			ctx = context.WithValue(ctx, "user_id", tokenInfo.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
