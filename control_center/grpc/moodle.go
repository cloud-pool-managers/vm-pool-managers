package grpc

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"control_center/config"
	"control_center/internal/moodle"
	"control_center/models"
)

// writeJSON est un petit helper de réponse JSON.
func writeJSONMoodle(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// GET /api/moodle/status — indique si Moodle est configuré (pour activer l'UI conditionnellement).
func handleMoodleStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{"configured": moodle.Configured()}
	if c, err := moodle.New(); err == nil {
		resp["url"] = c.BaseHost()
	}
	writeJSONMoodle(w, http.StatusOK, resp)
}

// GET /api/moodle/courses — liste les cours Moodle (pour le sélecteur d'import).
func handleMoodleCourses(w http.ResponseWriter, r *http.Request) {
	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	courses, err := c.GetCourses()
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"courses": courses})
}

type moodleStudentDTO struct {
	MoodleID  int    `json:"moodle_id"`
	Email     string `json:"email"`
	FullName  string `json:"fullname"`
	IsTeacher bool   `json:"is_teacher"`
}

// GET /api/moodle/enrolments?course_id=X — élèves inscrits (aperçu avant import).
func handleMoodleEnrolments(w http.ResponseWriter, r *http.Request) {
	courseID, err := strconv.Atoi(r.URL.Query().Get("course_id"))
	if err != nil || courseID <= 0 {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "course_id invalide"})
		return
	}
	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	users, err := c.GetEnrolledUsers(courseID)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	out := make([]moodleStudentDTO, 0, len(users))
	for _, u := range users {
		out = append(out, moodleStudentDTO{
			MoodleID: u.ID, Email: u.Email, FullName: u.FullName, IsTeacher: u.IsTeacher(),
		})
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"students": out})
}

type moodleImportRequest struct {
	PoolID   string   `json:"pool_id"`
	UserID   string   `json:"user_id"`
	CourseID int      `json:"course_id"`
	Emails   []string `json:"emails"` // optionnel : restreint l'import à ces emails
}

// POST /api/moodle/import — importe les élèves d'un cours Moodle dans un pool.
// Crée une ligne students par élève (Name = email = id nbgrader, MoodleEmail, MoodleUserID),
// sans clé SSH (l'accès se fait via JupyterLab/Guacamole ; clé ajoutable plus tard).
func handleMoodleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req moodleImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "JSON invalide"})
		return
	}
	if req.PoolID == "" || req.UserID == "" || req.CourseID <= 0 {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "pool_id, user_id et course_id requis"})
		return
	}

	c, err := moodle.New()
	if err != nil {
		writeJSONMoodle(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	users, err := c.GetEnrolledUsers(req.CourseID)
	if err != nil {
		writeJSONMoodle(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	// Filtre optionnel par emails sélectionnés.
	var only map[string]bool
	if len(req.Emails) > 0 {
		only = map[string]bool{}
		for _, e := range req.Emails {
			only[strings.ToLower(strings.TrimSpace(e))] = true
		}
	}

	// Pool + liste d'étudiants.
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

	imported, skipped := 0, 0
	for _, u := range users {
		if u.IsTeacher() || u.Email == "" {
			continue
		}
		key := strings.ToLower(u.Email)
		if only != nil && !only[key] {
			continue
		}
		if existing[key] {
			skipped++
			continue
		}
		student := models.Student{
			ListId:       list.ID,
			Name:         u.Email, // = identifiant nbgrader
			MoodleEmail:  u.Email,
			MoodleUserID: u.ID,
		}
		if err := config.Database.Create(&student).Error; err != nil {
			writeJSONMoodle(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		existing[key] = true
		imported++
	}

	// Mémorise le lien pool ↔ cours Moodle (pour le push de notes).
	config.Database.Model(&models.Serverpool{}).Where("id = ?", pool.ID).
		Update("moodle_course_id", req.CourseID)

	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"imported": imported, "skipped": skipped, "course_id": req.CourseID,
	})
}
