package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/thanhhaudev/go-open-api/app/common"
	appErr "github.com/thanhhaudev/go-open-api/app/error"
	"github.com/thanhhaudev/go-open-api/app/model"
	"github.com/thanhhaudev/go-open-api/app/repository"
)

type (
	TenantService interface {
		GetRefreshToken(ctx context.Context, key string, secret string) (map[string]interface{}, error)
		GetAccessToken(ctx context.Context, refreshToken string) (map[string]interface{}, error)
		RefreshAccessToken(ctx context.Context, accessToken string) (map[string]interface{}, error)
	}

	tenantService struct {
		tenantRepository repository.TenantRepository
		redisClient      *redis.Client
		logger           *logrus.Logger
	}
)

// RefreshAccessToken retrieves a new access token
func (s *tenantService) RefreshAccessToken(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	s.logger.WithFields(logrus.Fields{
		"accessToken": accessToken,
	}).Info("RefreshAccessToken called")

	if len(accessToken) == 0 {
		return nil, &appErr.AuthError{
			Message: "Invalid access token",
			Code:    http.StatusBadRequest,
		}
	}

	// Retrieve the API key from Redis
	apiKey, err := s.redisClient.Get(ctx, fmt.Sprintf("%s.%s", common.AuthAccessTokenPrefix, accessToken)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, &appErr.AuthError{
				Message: "Invalid access token",
				Code:    http.StatusBadRequest,
			}
		}

		s.logger.WithError(err).Error("Failed to get API key from Redis")

		return nil, &appErr.AuthError{
			Message: "Internal server error",
			Code:    http.StatusInternalServerError,
		}
	}

	tenant, err := s.tenantRepository.FindByApiKey(apiKey)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find tenant by API key")

		return nil, &appErr.AuthError{
			Message: "Internal server error",
			Code:    http.StatusInternalServerError,
		}
	}

	// verify the access token
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(tenant.ApiSecret), nil
	},
		jwt.WithAudience("tenant"),
		jwt.WithIssuer("localhost"),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		s.logger.WithError(err).Error("Failed to parse access token")

		return nil, &appErr.AuthError{
			Message: "Internal server error",
			Code:    http.StatusBadRequest,
		}
	}

	if !token.Valid {
		s.logger.Error("Invalid access token")

		return nil, &appErr.AuthError{
			Message: "Invalid access token",
			Code:    http.StatusBadRequest,
		}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		s.logger.Error("Failed to get claims from access token")

		return nil, &appErr.AuthError{
			Message: "Internal server error",
			Code:    http.StatusInternalServerError,
		}
	}

	claims["exp"] = time.Now().Unix() + common.AuthAccessTokenExpire
	claims["scopes"] = tenant.GetScopes()
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newAccessToken, err := token.SignedString([]byte(tenant.ApiSecret))
	if err != nil {
		s.logger.WithError(err).Error("Failed to build access token")

		return nil, &appErr.AuthError{
			Message: "Internal server error",
			Code:    http.StatusInternalServerError,
		}
	}

	// Save the access token to Redis
	s.redisClient.Set(ctx, fmt.Sprintf("%s.%s", common.AuthAccessTokenPrefix, newAccessToken), tenant.ApiKey, time.Duration(common.AuthAccessTokenExpire)*time.Second)

	// Delete the old access token
	s.redisClient.Del(ctx, fmt.Sprintf("%s.%s", common.AuthAccessTokenPrefix, accessToken))

	return map[string]interface{}{
		"access_token": newAccessToken,
		"expires_in":   common.AuthAccessTokenExpire,
		"scopes":       tenant.GetScopes(),
	}, nil
}

// GetAccessToken gets an access token
func (s *tenantService) GetAccessToken(ctx context.Context, refreshToken string) (map[string]interface{}, error) {
	s.logger.WithFields(logrus.Fields{
		"refreshToken": refreshToken,
	}).Info("GetAccessToken called")

	if len(refreshToken) == 0 {
		return nil, &appErr.AuthError{
			Message: "Invalid refresh token",
			Code:    http.StatusBadRequest,
		}
	}

	// Retrieve the API key from Redis
	apiKey, err := s.redisClient.Get(ctx, fmt.Sprintf("%s.%s", common.AuthRefreshTokenPrefix, refreshToken)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, &appErr.AuthError{
				Message: "Invalid refresh token",
				Code:    http.StatusBadRequest,
			}
		}

		s.logger.WithError(err).Error("Failed to get API key from Redis")

		return nil, &appErr.AuthError{
			Message: "Internal server error",
			Code:    http.StatusInternalServerError,
		}
	}

	tenant, err := s.tenantRepository.FindByApiKey(apiKey)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find tenant by API key")

		return nil, &appErr.AuthError{
			Message: "Internal server error",
			Code:    http.StatusInternalServerError,
		}
	}

	// verify the refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(tenant.ApiSecret), nil
	},
		jwt.WithAudience("tenant"),
		jwt.WithIssuer("localhost"),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		s.logger.WithError(err).Error("Failed to parse refresh token")

		return nil, &appErr.AuthError{
			Message: "Internal server error",
			Code:    http.StatusBadRequest,
		}
	}

	if !token.Valid {
		s.logger.Error("Invalid refresh token")

		return nil, &appErr.AuthError{
			Message: "Invalid refresh token",
			Code:    http.StatusBadRequest,
		}
	}

	expiresIn := common.AuthAccessTokenExpire // 2 days
	accessToken, err := buildToken(tenant, expiresIn)
	if err != nil {
		s.logger.WithError(err).Error("Failed to build access token")

		return nil, err
	}

	// Save the access token to Redis
	s.redisClient.Set(ctx, fmt.Sprintf("%s.%s", common.AuthAccessTokenPrefix, accessToken), tenant.ApiKey, time.Duration(expiresIn)*time.Second)

	return map[string]interface{}{
		"access_token": accessToken,
		"expires_in":   expiresIn,
		"scopes":       tenant.GetScopes(),
	}, nil
}

// GetRefreshToken gets an access token
func (s *tenantService) GetRefreshToken(ctx context.Context, key string, secret string) (map[string]interface{}, error) {
	tenant, err := s.tenantRepository.Find(key, secret)
	if err != nil {
		return nil, &appErr.AuthError{
			Message: "Invalid API key or secret",
			Code:    http.StatusBadRequest,
		}
	}

	expiresIn := common.AuthRefreshTokenExpire
	refreshToken, err := buildToken(tenant, expiresIn)
	if err != nil {
		s.logger.WithError(err).Error("Failed to build access token")

		return nil, err
	}

	// save the refresh token to Redis
	// key: refresh_token.<token string>, value: apiKey
	s.redisClient.Set(ctx, fmt.Sprintf("%s.%s", common.AuthRefreshTokenPrefix, refreshToken), tenant.ApiKey, time.Duration(expiresIn)*time.Second)

	return map[string]interface{}{
		"refresh_token": refreshToken,
		"expires_in":    expiresIn,
	}, nil
}

// buildToken builds a token
func buildToken(tenant *model.Tenant, e int64) (string, error) {
	now := time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})
	claims := token.Claims.(jwt.MapClaims) // refer to https://datatracker.ietf.org/doc/html/rfc7519#section-4.1 for more details
	claims["iss"] = "localhost"
	claims["aud"] = "tenant"
	claims["sub"] = tenant.ID
	claims["iat"] = now
	claims["nbf"] = now
	claims["exp"] = now + e // 7 days
	claims["scopes"] = tenant.GetScopes()

	refreshToken, err := token.SignedString([]byte(tenant.ApiSecret))
	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

// NewTenantService creates a new TenantService
func NewTenantService(r repository.TenantRepository, s *redis.Client, l *logrus.Logger) TenantService {
	return &tenantService{
		tenantRepository: r,
		redisClient:      s,
		logger:           l,
	}
}
