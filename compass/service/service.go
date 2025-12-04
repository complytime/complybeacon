package service

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/gin-contrib/requestid"
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
		slog.Warn("invalid enrichment request",
			slog.String("request_id", requestid.Get(c)),
			slog.String("error", err.Error()),
		)
		sendCompassError(c, http.StatusBadRequest, "Invalid format for enrichment")
		return
	}

	slog.Debug("enrich request received",
		slog.String("request_id", requestid.Get(c)),
		slog.String("policy_rule_id", req.Policy.PolicyRuleId),
		slog.String("policy_engine_name", req.Policy.PolicyEngineName),
	)

	mapperPlugin, ok := s.set[mapper.ID(req.Policy.PolicyEngineName)]
	if !ok {
		// Use fallback
		log.Printf("WARNING: Policy engine %s not found in mapper set, using basic mapper fallback", req.Policy.PolicyEngineName)
		mapperPlugin = basic.NewBasicMapper()
	}

	slog.Debug("mapper selected",
		slog.String("request_id", requestid.Get(c)),
		slog.String("mapper_id", string(mapperPlugin.PluginName())),
		slog.Bool("fallback_used", !ok),
	)

	compliance := mapperPlugin.Map(req.Policy, s.scope)
	enrichedResponse := api.EnrichmentResponse{
		Compliance: compliance,
	}

	slog.Debug("enrich result",
		slog.String("request_id", requestid.Get(c)),
		slog.String("mapping_status", string(enrichedResponse.Compliance.EnrichmentStatus)),
		slog.String("compliance_catalog", enrichedResponse.Compliance.Control.CatalogId),
		slog.String("compliance_control", enrichedResponse.Compliance.Control.Id),
	)

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
