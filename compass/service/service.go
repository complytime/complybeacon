package service

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/complytime/complybeacon/compass/api"

	"github.com/complytime/complybeacon/compass/mapper"
	"github.com/complytime/complybeacon/compass/mapper/plugins/basic"
)

// Service struct to hold dependencies if needed
type Service struct {
	set     mapper.Set
	scope   mapper.Scope
	version string
}

// NewService initializes a new Service instance.
func NewService(transformers mapper.Set, scope mapper.Scope) *Service {
	return &Service{
		set:     transformers,
		scope:   scope,
		version: "1.0.0",
	}
}

// PostV1Enrich handles the POST /v1/enrich endpoint.
// It's a handler function for Gin.
func (s *Service) PostV1Enrich(c *gin.Context) {
	var req api.EnrichmentRequest
	err := c.Bind(&req)
	if err != nil {
		sendCompassError(c, http.StatusBadRequest, "Invalid format for enrichment")
		return
	}

	mapperPlugin, ok := s.set[mapper.ID(req.Policy.PolicyEngineName)]
	if !ok {
		// Use fallback
		log.Printf("WARNING: Policy engine %s not found in mapper set, using basic mapper fallback", req.Policy.PolicyEngineName)
		mapperPlugin = basic.NewBasicMapper()
	}
	compliance := mapperPlugin.Map(req.Policy, s.scope)
	enrichedResponse := api.EnrichmentResponse{
		Compliance: compliance,
	}

	c.JSON(http.StatusOK, enrichedResponse)
}

// sendCompassError wraps sending of an error in the Error format, and
// handling the failure to marshal that.
func sendCompassError(c *gin.Context, code int32, message string) {
	compassErr := api.Error{
		Code:    code,
		Message: message,
	}
	c.JSON(int(code), compassErr)
}
