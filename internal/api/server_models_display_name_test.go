package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
)

func TestModelsEndpoint_DisplayName(t *testing.T) {
	server := newTestServer(t)
	const clientID = "openai-display-name-test-client"
	models := []*registry.ModelInfo{
		{
			ID:                  "display-name-configured-model",
			DisplayName:         "  Configured Model / 日本語  ",
			Created:             1700000000,
			OwnedBy:             "test-provider",
			Type:                "internal-provider-type",
			Version:             "internal-version",
			Description:         "Internal model description",
			ContextLength:       32768,
			MaxContextLength:    65536,
			MaxCompletionTokens: 4096,
			SupportedParameters: []string{"temperature"},
			MetadataModelID:     "internal-upstream-model",
		},
		{ID: "display-name-missing-model", OwnedBy: "test-provider"},
		{ID: "display-name-empty-model", DisplayName: "", OwnedBy: "test-provider"},
	}
	modelRegistry := registry.GetGlobalRegistry()
	modelRegistry.RegisterClient(clientID, "openai-compatibility", models)
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	recorder := httptest.NewRecorder()
	server.engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response map[string]json.RawMessage
	if errUnmarshal := json.Unmarshal(recorder.Body.Bytes(), &response); errUnmarshal != nil {
		t.Fatalf("decode response: %v", errUnmarshal)
	}
	if len(response) != 2 || string(response["object"]) != `"list"` {
		t.Fatalf("unexpected list envelope: %s", recorder.Body.String())
	}
	var entries []map[string]any
	if errUnmarshal := json.Unmarshal(response["data"], &entries); errUnmarshal != nil {
		t.Fatalf("decode model list: %v", errUnmarshal)
	}
	byID := make(map[string]map[string]any, len(entries))
	for _, entry := range entries {
		id, _ := entry["id"].(string)
		byID[id] = entry
	}
	for _, model := range models {
		t.Run(model.ID, func(t *testing.T) {
			want := map[string]any{
				"id":       model.ID,
				"object":   "model",
				"owned_by": model.OwnedBy,
			}
			if model.Created > 0 {
				want["created"] = float64(model.Created)
			}
			if model.DisplayName != "" {
				want["display_name"] = model.DisplayName
			}
			if got := byID[model.ID]; !reflect.DeepEqual(got, want) {
				t.Errorf("model = %#v, want exactly %#v", got, want)
			}
		})
	}
}
