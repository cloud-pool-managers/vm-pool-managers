package grpc

import (
	"context"
	"net/http"
	"strings"

	"control_center/config"
	"control_center/models"

	"github.com/danielgtaylor/huma/v2"
)

// registerImageProposalsHuma stocke et liste les propositions d'images des enseignants.
//
//	POST /api/image-proposals  {github_url, name, description, submitted_by}  → 201 + proposal
//	GET  /api/image-proposals?user=<email>  → propositions (récentes d'abord)
func registerImageProposalsHuma(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-image-proposal", Method: http.MethodPost, Path: "/api/image-proposals",
		Summary: "Proposer une image", Tags: []string{"images"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		Body struct {
			GithubURL   string `json:"github_url"`
			Name        string `json:"name"`
			Description string `json:"description"`
			SubmittedBy string `json:"submitted_by"`
		}
	}) (*AnyOutput, error) {
		body := in.Body
		body.GithubURL = strings.TrimSpace(body.GithubURL)
		body.Name = strings.TrimSpace(body.Name)
		if body.GithubURL == "" || body.Name == "" {
			return nil, huma.Error400BadRequest("github_url et name sont requis")
		}
		if !strings.HasPrefix(body.GithubURL, "https://github.com/") &&
			!strings.HasPrefix(body.GithubURL, "http://github.com/") {
			return nil, huma.Error400BadRequest("github_url doit être une URL github.com")
		}

		p := models.ImageProposal{
			GithubURL:   body.GithubURL,
			Name:        body.Name,
			Description: strings.TrimSpace(body.Description),
			SubmittedBy: strings.TrimSpace(body.SubmittedBy),
			Status:      "pending",
		}
		if err := config.Database.Create(&p).Error; err != nil {
			return nil, huma.Error500InternalServerError("db error: " + err.Error())
		}
		return &AnyOutput{Body: p}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-image-proposals", Method: http.MethodGet, Path: "/api/image-proposals",
		Summary: "Lister les propositions d'images", Tags: []string{"images"},
	}, func(ctx context.Context, in *struct {
		User string `query:"user"`
	}) (*AnyOutput, error) {
		var list []models.ImageProposal
		q := config.Database.Order("created_at DESC")
		if in.User != "" {
			q = q.Where("submitted_by = ?", in.User)
		}
		q.Find(&list)
		return &AnyOutput{Body: list}, nil
	})
}
