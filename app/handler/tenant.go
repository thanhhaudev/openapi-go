package handler

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/thanhhaudev/go-open-api/app/command"
	"github.com/thanhhaudev/go-open-api/app/config"
	"github.com/thanhhaudev/go-open-api/app/repository"
	"github.com/thanhhaudev/go-open-api/app/service"
	"github.com/thanhhaudev/go-open-api/app/util"
)

type tenantHandler struct {
	tenantService service.TenantService
	logger        *logrus.Logger
}

// RefreshAccessToken	godoc
// @Summary      		Retrieve a new access token
// @Tags         		auth
// @Accept       		json
// @Produce      		json
// @Param				request body command.RefreshTokenRequest true "request body"
// @Success      		200  {object} map[string]interface{}
// @Failure      		400  {object} error.AuthError
// @Failure      		500  {object} error.AuthError
// @Router       		/api/auth/refresh [post]
func (t tenantHandler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	p := command.RefreshTokenRequest{}
	err := util.Bind(r, &p)
	if err != nil {
		t.logger.WithError(err).Error("Failed to parse request body")
		util.Response(w, err.Error(), http.StatusBadRequest)

		return
	}

	data, err := t.tenantService.RefreshAccessToken(r.Context(), p.AccessToken)
	if err != nil {
		util.Response(w, err, http.StatusBadRequest)
		return
	}

	util.Response(w, map[string]interface{}{"code": http.StatusOK, "data": data}, http.StatusOK)
}

// GetAccessToken	godoc
// @Summary      	Exchange refresh token for access token
// @Tags         	auth
// @Accept       	json
// @Produce      	json
// @Param			request body command.ExchangeTokenRequest true "request body"
// @Success      	200  {object} map[string]interface{}
// @Failure      	400  {object} error.AuthError
// @Failure      	500  {object} error.AuthError
// @Router       	/api/auth/exchange [post]
func (t tenantHandler) GetAccessToken(w http.ResponseWriter, r *http.Request) {
	p := command.ExchangeTokenRequest{}
	err := util.Bind(r, &p)
	if err != nil {
		t.logger.WithError(err).Error("Failed to parse request body")
		util.Response(w, err.Error(), http.StatusBadRequest)

		return
	}

	data, err := t.tenantService.GetAccessToken(r.Context(), p.RefreshToken)
	if err != nil {
		util.Response(w, err, http.StatusBadRequest)
		return
	}

	util.Response(w, map[string]interface{}{"code": http.StatusOK, "data": data}, http.StatusOK)
}

// GetRefreshToken	godoc
// @Summary      	Retrieve refresh token using API key and secret
// @Tags         	auth
// @Accept       	json
// @Produce      	json
// @Param			request body command.AccessTokenRequest true "request body"
// @Success      	200  {object} map[string]interface{}
// @Failure      	400  {object} error.AuthError
// @Failure      	500  {object} error.AuthError
// @Router       	/api/auth/access [post]
func (t tenantHandler) GetRefreshToken(w http.ResponseWriter, r *http.Request) {
	p := command.AccessTokenRequest{}
	err := util.Bind(r, &p)
	if err != nil {
		t.logger.WithError(err).Error("Failed to parse request body")
		util.Response(w, err.Error(), http.StatusBadRequest)

		return
	}

	data, err := t.tenantService.GetRefreshToken(r.Context(), p.ApiKey, p.ApiSecret)
	if err != nil {
		t.logger.WithError(err).Error("Failed to get refresh token")
		util.Response(w, err, http.StatusBadRequest)

		return
	}

	util.Response(w, map[string]interface{}{"code": http.StatusOK, "data": data}, http.StatusOK)
}

// NewTenantHandler creates a new TenantHandler
func NewTenantHandler(r repository.TenantRepository, l *logrus.Logger, s *config.RedisStore) TenantHandler {
	return &tenantHandler{
		tenantService: service.NewTenantService(r, s.Client, l),
		logger:        l,
	}
}
