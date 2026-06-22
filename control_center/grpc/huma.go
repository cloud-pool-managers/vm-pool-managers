package grpc

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// newHumaAPI monte une API HUMA sur le mux REST existant. Les chemins /api/* et le
// middleware d'auth en amont (httpAuthMiddleware) sont inchangés ; les endpoints non
// encore migrés restent en mux.HandleFunc et coexistent (migration incrémentale).
// Fournit en plus l'OpenAPI 3.1 (/api/openapi.json|yaml) et la doc (/api/docs).
func newHumaAPI(mux *http.ServeMux) huma.API {
	config := huma.DefaultConfig("CloudPoolManager API", "1.0.0")
	config.OpenAPIPath = "/api/openapi"
	config.DocsPath = "/api/docs"
	api := humago.New(mux, config)
	registerHumaRoutes(api)
	return api
}

// registerHumaRoutes enregistre les opérations migrées vers HUMA.
// On y déplace les endpoints au fur et à mesure (et on retire le mux.HandleFunc correspondant).
func registerHumaRoutes(api huma.API) {
	// GET /api/me — identité + rôle effectif de l'appelant.
	huma.Register(api, huma.Operation{
		OperationID: "get-me",
		Method:      http.MethodGet,
		Path:        "/api/me",
		Summary:     "Identité et rôle de l'appelant",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, _ *struct{}) (*MeOutput, error) {
		id, ok := identityFrom(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("non authentifié")
		}
		out := &MeOutput{}
		out.Body.Email = id.Email
		out.Body.Role = id.Role
		out.Body.IsAdmin = id.Role == RoleAdmin
		out.Body.IsStaff = isStaff(id.Role)
		out.Body.Via = id.Via
		return out, nil
	})
}

// MeOutput : réponse de GET /api/me (forme JSON identique à l'ancien handler).
type MeOutput struct {
	Body struct {
		Email   string `json:"email"`
		Role    string `json:"role"`
		IsAdmin bool   `json:"is_admin"`
		IsStaff bool   `json:"is_staff"`
		Via     string `json:"via"`
	}
}
