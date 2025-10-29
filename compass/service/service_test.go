package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/complytime/complybeacon/compass/api"
	"github.com/complytime/complybeacon/compass/mapper"
)

func TestNewService(t *testing.T) {
	mappers := make(mapper.Set)
	scope := make(mapper.Scope)

	service := NewService(mappers, scope)

	assert.NotNil(t, service)
	assert.Equal(t, mappers, service.set)
	assert.Equal(t, scope, service.scope)
	assert.Equal(t, "1.0.0", service.version)
}

func TestPostV1Metadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success/HappyPath", func(t *testing.T) {
		// Setup
		mappers := make(mapper.Set)
		scope := make(mapper.Scope)
		service := NewService(mappers, scope)

		// Create test request
		request := api.MetadataRequest{
			Policy: api.Policy{
				PolicyEngineName: "test-engine",
				PolicyRuleId:     "AC-1",
			},
		}

		// Setup router
		router := gin.New()
		router.POST("/v1/metadata", service.PostV1Metadata)

		// Create request body
		reqBody, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/v1/metadata", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.MetadataResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response.Compliance)
	})

	t.Run("Failure/InvalidRequest", func(t *testing.T) {
		// Setup
		mappers := make(mapper.Set)
		scope := make(mapper.Scope)
		service := NewService(mappers, scope)

		// Setup router
		router := gin.New()
		router.POST("/v1/metadata", service.PostV1Metadata)

		// Create invalid request
		req := httptest.NewRequest("POST", "/v1/metadata", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success/UsesFallback", func(t *testing.T) {
		// Setup with empty mappers to trigger fallback
		mappers := make(mapper.Set)
		scope := make(mapper.Scope)
		service := NewService(mappers, scope)

		// Create test request with unknown policy engine
		request := api.MetadataRequest{
			Policy: api.Policy{
				PolicyEngineName: "unknown-engine",
				PolicyRuleId:     "AC-1",
			},
		}

		// Setup router
		router := gin.New()
		router.POST("/v1/metadata", service.PostV1Metadata)

		// Create request body
		reqBody, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/v1/metadata", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response api.MetadataResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response.Compliance)
	})
}

func TestSendCompassError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test-error", func(c *gin.Context) {
		sendCompassError(c, http.StatusInternalServerError, "Test error message")
	})

	// Create request
	req := httptest.NewRequest("GET", "/test-error", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse api.Error
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, int32(http.StatusInternalServerError), errorResponse.Code)
	assert.Equal(t, "Test error message", errorResponse.Message)
}
