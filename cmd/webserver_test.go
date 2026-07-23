package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayswind/financially_free/pkg/api"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/middlewares"
	"github.com/mayswind/financially_free/pkg/settings"
)

func TestAssetPlanRouteRejectsUnauthenticatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1Route := router.Group("/api/v1")
	apiV1Route.Use(bindMiddleware(middlewares.JWTAuthorization(&settings.Config{})))
	apiV1Route.GET("/asset_plans/get.json", bindApi(api.AssetPlans.AssetPlanGetHandler))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/asset_plans/get.json", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, float64(errs.ErrTokenIsEmpty.Code()), body["errorCode"])
}
