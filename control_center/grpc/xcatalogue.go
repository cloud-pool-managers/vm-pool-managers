package grpc

import (
	"encoding/json"
	"net/http"
	"strings"

	"control_center/config"
	"control_center/internal/xcatalogue"
	"control_center/models"
)

// GET /api/xcours/status — indique si les endpoints PROTÉGÉS (affectations) sont utilisables
// (token présent). Le catalogue, lui, est public et marche toujours.
func handleXCoursStatus(w http.ResponseWriter, r *http.Request) {
	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"catalogue_available":    true,                    // public
		"affectations_available": xcatalogue.Configured(), // nécessite XCOURSES_TOKEN
	})
}

// GET /api/xcours/catalogue?year=&dep= — catalogue public des cours de l'X.
func handleXCoursCatalogue(w http.ResponseWriter, r *http.Request) {
	c := xcatalogue.New()
	courses, err := c.Catalogue(r.URL.Query().Get("year"), r.URL.Query().Get("dep"))
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"courses": courses})
}

// GET /api/xcours/members?id=CODE_EP-ANNÉE — profs + élèves d'un cours (token requis).
func handleXCoursMembers(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "id requis"})
		return
	}
	members, err := xcatalogue.New().CourseMembers(id)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	teachers, students := 0, 0
	for _, m := range members {
		if m.IsTeacher() {
			teachers++
		} else {
			students++
		}
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"members": members, "teachers": teachers, "students": students,
	})
}

// GET /api/xcours/groups?id=CODE_EP-ANNÉE — groupes (TD/PC) d'un cours (token requis).
func handleXCoursGroups(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "id requis"})
		return
	}
	groups, err := xcatalogue.New().CourseGroups(id)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"groups": groups})
}

type xcoursImportRequest struct {
	PoolID     string   `json:"pool_id"`
	UserID     string   `json:"user_id"`
	CourseCode string   `json:"course_code"` // id du cours (shortname)
	GroupName  string   `json:"group_name"`  // optionnel : restreint à un groupe
	Usernames  []string `json:"usernames"`   // optionnel : restreint à ces logins
}

// POST /api/xcours/import — importe les élèves d'un cours de l'X dans un pool.
// Calqué sur l'import Moodle : une ligne students par élève (Name = MoodleEmail = login,
// = id nbgrader), accès via login établissement (attribution sans clé SSH).
func handleXCoursImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req xcoursImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "JSON invalide"})
		return
	}
	if req.PoolID == "" || req.UserID == "" || req.CourseCode == "" {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "pool_id, user_id et course_code requis"})
		return
	}

	c := xcatalogue.New()
	members, err := c.CourseMembers(req.CourseCode)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	// Filtre optionnel par groupe (TD/PC).
	var groupOnly map[string]bool
	if req.GroupName != "" {
		groups, err := c.CourseGroups(req.CourseCode)
		if err != nil {
			writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
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
		writeJSONMoodle(w, http.StatusNotFound, map[string]string{"error": "pool introuvable"})
		return
	}
	list := &pool.ListStudents
	if list.ID == 0 {
		list.PoolId = pool.ID
		if err := config.Database.Create(list).Error; err != nil {
			writeJSONMoodle(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
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
			writeJSONMoodle(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		existing[key] = true
		imported++
	}

	// Mémorise le lien pool ↔ cours de l'X.
	config.Database.Model(&models.Serverpool{}).Where("id = ?", pool.ID).
		Update("x_course_code", req.CourseCode)

	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"imported": imported, "skipped": skipped, "course_code": req.CourseCode,
	})
}
