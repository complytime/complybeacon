package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewCacheableClient(t *testing.T) {
	baseClient, err := NewClient("http://localhost:8080")
	assert.NoError(t, err)

	cacheableClient := NewCacheableClient(baseClient, zap.NewNop(), 0)
	assert.NotNil(t, cacheableClient)
	assert.Equal(t, baseClient, cacheableClient.client)
	assert.NotNil(t, cacheableClient.cache)
	assert.NotNil(t, cacheableClient.logger)
}

func TestCacheableClient_Retrieve(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() (*httptest.Server, *int)
		policy      Policy
		calls       []struct {
			policy            Policy
			expectedError     bool
			expectedErrorMsg  string
			expectedControlId string
			expectedApiCalls  int
		}
	}{
		{
			name:        "Success/BasicRetrieval",
			setupServer: workingServer,
			policy: Policy{
				PolicyRuleId:     "test-policy-123",
				PolicyEngineName: "test-engine",
			},
			calls: []struct {
				policy            Policy
				expectedError     bool
				expectedErrorMsg  string
				expectedControlId string
				expectedApiCalls  int
			}{
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:     false,
					expectedControlId: "OSPS-QA-01.01",
					expectedApiCalls:  1,
				},
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:     false,
					expectedControlId: "OSPS-QA-01.01",
					expectedApiCalls:  1,
				},
			},
		},
		{
			name:        "Success/CacheHit",
			setupServer: workingServer,
			policy: Policy{
				PolicyRuleId:     "test-policy-123",
				PolicyEngineName: "test-engine",
			},
			calls: []struct {
				policy            Policy
				expectedError     bool
				expectedErrorMsg  string
				expectedControlId string
				expectedApiCalls  int
			}{
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:     false,
					expectedControlId: "OSPS-QA-01.01",
					expectedApiCalls:  1,
				},
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:     false,
					expectedControlId: "OSPS-QA-01.01",
					expectedApiCalls:  1,
				},
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:     false,
					expectedControlId: "OSPS-QA-01.01",
					expectedApiCalls:  1,
				},
			},
		},
		{
			name:        "Success/CacheMiss",
			setupServer: workingServer,
			policy: Policy{
				PolicyRuleId:     "test-policy-123",
				PolicyEngineName: "test-engine",
			},
			calls: []struct {
				policy            Policy
				expectedError     bool
				expectedErrorMsg  string
				expectedControlId string
				expectedApiCalls  int
			}{
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:     false,
					expectedControlId: "OSPS-QA-01.01",
					expectedApiCalls:  1,
				},
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-456",
						PolicyEngineName: "test-engine",
					},
					expectedError:     false,
					expectedControlId: "OSPS-QA-01.01",
					expectedApiCalls:  2,
				},
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:     false,
					expectedControlId: "OSPS-QA-01.01",
					expectedApiCalls:  2,
				},
			},
		},
		{
			name: "Failure/APIError",
			setupServer: func() (*httptest.Server, *int) {
				apiCallCount := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					apiCallCount++
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"code": 500, "message": "Internal server error"}`))
				}))
				return server, &apiCallCount
			},
			policy: Policy{
				PolicyRuleId:     "test-policy-123",
				PolicyEngineName: "test-engine",
			},
			calls: []struct {
				policy            Policy
				expectedError     bool
				expectedErrorMsg  string
				expectedControlId string
				expectedApiCalls  int
			}{
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:    true,
					expectedErrorMsg: "failed to fetch metadata",
					expectedApiCalls: 1,
				},
			},
		},
		{
			name: "Failure/ResponseError",
			setupServer: func() (*httptest.Server, *int) {
				apiCallCount := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					apiCallCount++
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`invalid json`))
				}))
				return server, &apiCallCount
			},
			policy: Policy{
				PolicyRuleId:     "test-policy-123",
				PolicyEngineName: "test-engine",
			},
			calls: []struct {
				policy            Policy
				expectedError     bool
				expectedErrorMsg  string
				expectedControlId string
				expectedApiCalls  int
			}{
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:    true,
					expectedErrorMsg: "failed to fetch metadata",
					expectedApiCalls: 1,
				},
			},
		},
		{
			name: "Failure/ServerUnavailable",
			setupServer: func() (*httptest.Server, *int) {
				apiCallCount := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					apiCallCount++
					w.WriteHeader(http.StatusOK)
				}))
				server.Close()
				return server, &apiCallCount
			},
			policy: Policy{
				PolicyRuleId:     "test-policy-123",
				PolicyEngineName: "test-engine",
			},
			calls: []struct {
				policy            Policy
				expectedError     bool
				expectedErrorMsg  string
				expectedControlId string
				expectedApiCalls  int
			}{
				{
					policy: Policy{
						PolicyRuleId:     "test-policy-123",
						PolicyEngineName: "test-engine",
					},
					expectedError:    true,
					expectedErrorMsg: "failed to fetch metadata",
					expectedApiCalls: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, apiCallCount := tt.setupServer()
			if server != nil {
				defer server.Close()
			}

			baseClient, err := NewClient(server.URL)
			require.NoError(t, err)

			cacheableClient := NewCacheableClient(baseClient, zap.NewNop(), 0)

			var firstCompliance Compliance
			for i, call := range tt.calls {
				expectedCountBefore := *apiCallCount
				compliance, err := cacheableClient.Retrieve(context.Background(), call.policy)
				actualCountAfter := *apiCallCount

				if call.expectedError {
					assert.Error(t, err)
					if call.expectedErrorMsg != "" {
						assert.Contains(t, err.Error(), call.expectedErrorMsg)
					}
					assert.Empty(t, compliance.Control.Id)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, call.expectedControlId, compliance.Control.Id)
					if i == 0 {
						firstCompliance = compliance
					} else if call.policy.PolicyRuleId == tt.calls[0].policy.PolicyRuleId {
						// Verify cached content matches when using the same policy
						assert.Equal(t, firstCompliance.Control.Id, compliance.Control.Id)
						assert.Equal(t, firstCompliance.Control.CatalogId, compliance.Control.CatalogId)
						assert.Equal(t, firstCompliance.EnrichmentStatus, compliance.EnrichmentStatus)
					}

					if call.expectedApiCalls > 0 {
						assert.Equal(t, call.expectedApiCalls, actualCountAfter)
					} else {
						// For cache hits, verify no new API calls were made
						if i > 0 && call.policy.PolicyRuleId == tt.calls[0].policy.PolicyRuleId {
							assert.Equal(t, expectedCountBefore, actualCountAfter)
						}
					}
				}
			}
		})
	}
}

func workingServer() (*httptest.Server, *int) {
	apiCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++
		response := `{
						"compliance": {
							"control": {
								"id": "OSPS-QA-01.01",
								"catalogId": "OSPS-B",
								"category": "Quality"
							},
							"frameworks": {
								"requirements": ["CC-B-1", "1.2b"],
								"frameworks": ["BPB", "CRA"]
							},
							"enrichmentStatus": "success"
						}
					}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	return server, &apiCallCount
}
