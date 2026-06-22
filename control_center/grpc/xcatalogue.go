package grpc

import (
	"context"
	"net/http"
	"strings"

	"control_center/config"
	"control_center/internal/xcatalogue"
	"control_center/models"

	"github.com/danielgtaylor/huma/v2"
)

// registerXCoursHuma enregistre les endpoints /api/xcours/* (cours de l'X / Synapses).
func registerXCoursHuma(api huma.API) {
	// GET /api/xcours/status — endpoints protégés (affectations) utilisables ? Catalogue toujours public.
	huma.Register(api, huma.Operation{
		OperationID: "xcours-status", Method: http.MethodGet, Path: "/api/xcours/status",
		Summary: "Disponibilité du catalogue et des affectations X", Tags: []string{"xcours"},
	}, func(ctx context.Context, _ *struct{}) (*AnyOutput, error) {
		return &AnyOutput{Body: map[string]any{
			"catalogue_available":    true,                    // public
			"affectations_available": xcatalogue.Configured(), // nécessite XCOURSES_TOKEN
		}}, nil
	})

	// GET /api/xcours/catalogue?year=&dep= — catalogue public des cours de l'X.
	huma.Register(api, huma.Operation{
		OperationID: "xcours-catalogue", Method: http.MethodGet, Path: "/api/xcours/catalogue",
		Summary: "Catalogue des cours de l'X", Tags: []string{"xcours"},
	}, func(ctx context.Context, in *struct {
		Year string `query:"year"`
		Dep  string `query:"dep"`
	}) (*AnyOutput, error) {
		courses, err := xcatalogue.New().Catalogue(in.Year, in.Dep)
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		return &AnyOutput{Body: map[string]any{"courses": courses}}, nil
	})

	// GET /api/xcours/members?id=CODE_EP-ANNÉE — profs + élèves d'un cours (token requis).
	huma.Register(api, huma.Operation{
		OperationID: "xcours-members", Method: http.MethodGet, Path: "/api/xcours/members",
		Summary: "Membres d'un cours de l'X", Tags: []string{"xcours"},
	}, func(ctx context.Context, in *struct {
		ID string `query:"id"`
	}) (*AnyOutput, error) {
		id := strings.TrimSpace(in.ID)
		if id == "" {
			return nil, huma.Error400BadRequest("id requis")
		}
		members, err := xcatalogue.New().CourseMembers(id)
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		teachers, students := 0, 0
		for _, m := range members {
			if m.IsTeacher() {
				teachers++
			} else {
				students++
			}
		}
		return &AnyOutput{Body: map[string]any{
			"members": members, "teachers": teachers, "students": students,
		}}, nil
	})

	// GET /api/xcours/groups?id=CODE_EP-ANNÉE — groupes (TD/PC) d'un cours (token requis).
	huma.Register(api, huma.Operation{
		OperationID: "xcours-groups", Method: http.MethodGet, Path: "/api/xcours/groups",
		Summary: "Groupes (TD/PC) d'un cours de l'X", Tags: []string{"xcours"},
	}, func(ctx context.Context, in *struct {
		ID string `query:"id"`
	}) (*AnyOutput, error) {
		id := strings.TrimSpace(in.ID)
		if id == "" {
			return nil, huma.Error400BadRequest("id requis")
		}
		groups, err := xcatalogue.New().CourseGroups(id)
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		return &AnyOutput{Body: map[string]any{"groups": groups}}, nil
	})

	// POST /api/xcours/import — importe les élèves d'un cours de l'X dans un pool.
	huma.Register(api, huma.Operation{
		OperationID: "xcours-import", Method: http.MethodPost, Path: "/api/xcours/import",
		Summary: "Importer les élèves d'un cours de l'X", Tags: []string{"xcours"},
	}, func(ctx context.Context, in *struct{ Body xcoursImportRequest }) (*AnyOutput, error) {
		return handleXCoursImport(in.Body)
	})
}

type xcoursImportRequest struct {
	PoolID     string   `json:"pool_id"`
	UserID     string   `json:"user_id"`
	CourseCode string   `json:"course_code"` // id du cours (shortname)
	GroupName  string   `json:"group_name"`  // optionnel : restreint à un groupe
	Usernames  []string `json:"usernames"`   // optionnel : restreint à ces logins
}

// handleXCoursImport importe les élèves d'un cours de l'X dans un pool.
// Calqué sur l'import Moodle : une ligne students par élève (Name = MoodleEmail = login,
// = id nbgrader), accès via login établissement (attribution sans clé SSH).
func handleXCoursImport(req xcoursImportRequest) (*AnyOutput, error) {
	if req.PoolID == "" || req.UserID == "" || req.CourseCode == "" {
		return nil, huma.Error400BadRequest("pool_id, user_id et course_code requis")
	}

	c := xcatalogue.New()
	members, err := c.CourseMembers(req.CourseCode)
	if err != nil {
		return nil, huma.Error502BadGateway(err.Error())
	}

	// Filtre optionnel par groupe (TD/PC).
	var groupOnly map[string]bool
	if req.GroupName != "" {
		groups, err := c.CourseGroups(req.CourseCode)
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		groupOnly = map[string]bool{}
		for _, g := range groups {
			if g.GroupName == req.GroupName {
				groupOnly[strings.ToLower(g.Username)] = true
			}
		}
	}

	// Filtre optionnel par logins sélectionnés.
	var only map[string]bool
	if len(req.Usernames) > 0 {
		only = map[string]bool{}
		for _, u := range req.Usernames {
			only[strings.ToLower(strings.TrimSpace(u))] = true
		}
	}

	var pool models.Serverpool
	if err := config.Database.Preload("ListStudents.Students").
		Where("serverpool_id = ? AND user_id = ?", req.PoolID, req.UserID).
		First(&pool).Error; err != nil {
		return nil, huma.Error404NotFound("pool introuvable")
	}
	list := &pool.ListStudents
	if list.ID == 0 {
		list.PoolId = pool.ID
		if err := config.Database.Create(list).Error; err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
	}
	existing := map[string]bool{}
	for _, s := range list.Students {
		if s.MoodleEmail != "" {
			existing[strings.ToLower(s.MoodleEmail)] = true
		}
	}

	// Promotion des enseignants du cours au rôle prof (accès équipe pédagogique).
	for _, m := range members {
		if m.IsTeacher() && m.Username != "" {
			_ = upsertUserRole(m.Username, RoleProf)
		}
	}

	imported, skipped := 0, 0
	for _, m := range members {
		if m.IsTeacher() || m.Username == "" {
			continue
		}
		key := strings.ToLower(m.Username)
		if only != nil && !only[key] {
			continue
		}
		if groupOnly != nil && !groupOnly[key] {
			continue
		}
		if existing[key] {
			skipped++
			continue
		}
		student := models.Student{
			ListId:      list.ID,
			Name:        m.Username, // = identifiant nbgrader
			MoodleEmail: m.Username, // clé de jointure login établissement ↔ étudiant
		}
		if err := config.Database.Create(&student).Error; err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		existing[key] = true
		imported++
	}

	// Mémorise le lien pool ↔ cours de l'X.
	config.Database.Model(&models.Serverpool{}).Where("id = ?", pool.ID).
		Update("x_course_code", req.CourseCode)

	return &AnyOutput{Body: map[string]any{
		"imported": imported, "skipped": skipped, "course_code": req.CourseCode,
	}}, nil
}
