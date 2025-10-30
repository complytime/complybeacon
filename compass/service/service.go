package service

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/complytime/complybeacon/compass/api"

	httpmw "github.com/complytime/complybeacon/compass/internal/middleware"
	"github.com/complytime/complybeacon/compass/mapper"
	"github.com/complytime/complybeacon/compass/mapper/plugins/basic"
)

// Service struct to hold dependencies if needed
type Service struct {
	set   mapper.Set
	scope mapper.Scope
}

// NewService initializes a new Service instance.
func NewService(transformers mapper.Set, scope mapper.Scope) *Service {
	return &Service{
		set:   transformers,
		scope: scope,
	}
}

// PostV1Enrich handles the POST /v1/enrich endpoint.
// It's a handler function for Gin.
func (s *Service) PostV1Enrich(c *gin.Context) {
	var req api.EnrichmentRequest
	err := c.Bind(&req)
	if err != nil {
		slog.Warn("invalid enrichment request",
			slog.String("request_id", httpmw.GetRequestID(c)),
			slog.String("error", err.Error()),
		)
		sendCompassError(c, http.StatusBadRequest, "Invalid format for enrichment")
		return
	}

	slog.Debug("enrich request received",
		slog.String("request_id", httpmw.GetRequestID(c)),
		slog.String("evidence_id", req.Evidence.Id),
		slog.String("policy_id", req.Evidence.PolicyId),
		slog.String("evidence_source", req.Evidence.Source),
		slog.String("timestamp", req.Evidence.Timestamp.String()),
	)

	mapperPlugin, ok := s.set[mapper.ID(req.Evidence.Source)]
	if !ok {
		// Use fallback
		mapperPlugin = basic.NewBasicMapper()
	}

	slog.Debug("mapper selected",
		slog.String("request_id", httpmw.GetRequestID(c)),
		slog.String("mapper_id", string(mapperPlugin.PluginName())),
		slog.Bool("fallback_used", !ok),
	)

	enrichedResponse := enrich(req.Evidence, mapperPlugin, s.scope)

	statusId := -1
	if enrichedResponse.Status.Id != nil {
		statusId = int(*enrichedResponse.Status.Id)
	}

	slog.Debug("enrich result",
		slog.String("request_id", httpmw.GetRequestID(c)),
		slog.String("status_title", string(enrichedResponse.Status.Title)),
		slog.Int("status_id", statusId),
		slog.String("compliance_catalog", enrichedResponse.Compliance.Catalog),
		slog.String("compliance_control", enrichedResponse.Compliance.Control),
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

// Enrich the raw evidence with risk attributes based on `gemara` semantics.
func enrich(rawEnv api.RawEvidence, attributeMapper mapper.Mapper, scope mapper.Scope) api.EnrichmentResponse {
	compliance, status := attributeMapper.Map(rawEnv, scope)
	return api.EnrichmentResponse{
		Compliance: compliance,
		Status:     status,
	}
}
