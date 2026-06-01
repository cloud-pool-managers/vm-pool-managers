package grpc

import (
	"bytes"
	"control_center/config"
	"control_center/internal/sshinject"
	"control_center/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// nbgraderSSHClient dials the instructor VM for a given pool and returns a connected SSH client.
// The caller is responsible for closing the client.
func nbgraderSSHClient(poolID, userID string) (*ssh.Client, error) {
	var pool models.Serverpool
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		First(&pool).Error; err != nil {
		return nil, fmt.Errorf("pool not found: %w", err)
	}

	// Find the instructor VM (there should be only one, locked to the instructor).
	var server models.Server
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		First(&server).Error; err != nil {
		return nil, fmt.Errorf("instructor VM not found: %w", err)
	}
	if server.IP_Address == "" {
		return nil, fmt.Errorf("instructor VM has no IP address")
	}

	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = os.Getenv("SSH_KEY_PATH")
	}
	signer, err := sshinject.LoadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load SSH key: %w", err)
	}

	cfg := sshinject.SshConfig("vmuser", signer)
	client, err := ssh.Dial("tcp", server.IP_Address+":22", cfg)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", server.IP_Address, err)
	}
	return client, nil
}

// dockerExec wraps a command to run inside the 'jupyter' Docker container as jovyan.
// Falls back to direct execution if Docker is not available.
func dockerExec(cmd string) string {
	return fmt.Sprintf(`sudo docker exec jupyter bash -c %s 2>&1 || bash -c %s 2>&1`,
		shellQuote(cmd), shellQuote(cmd))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// runSSHOutput runs a command via SSH and returns its stdout as a string.
func runSSHOutput(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("run %q: %w (stderr: %s)", cmd, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// handleNbgraderAssignments lists released assignments on the instructor VM.
// GET /api/nbgrader/assignments?pool_id=X&user_id=Y
func handleNbgraderAssignments(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	if poolID == "" || userID == "" {
		http.Error(w, "missing pool_id or user_id", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		log.Printf("[nbgrader] assignments SSH error: %v", err)
		http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer client.Close()

	// List directories in nbgrader/source/ inside the jupyter container
	out, err := runSSHOutput(client, dockerExec(`ls -1 /home/jovyan/nbgrader/source/ 2>/dev/null || ls -1 /home/jovyan/ 2>/dev/null | grep -v work || echo ""`))
	if err != nil {
		http.Error(w, "SSH command failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var assignments []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			assignments = append(assignments, line)
		}
	}
	if assignments == nil {
		assignments = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"assignments": assignments})
}

// handleNbgraderCollect runs `nbgrader collect` on the instructor VM.
// POST /api/nbgrader/collect?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" {
		http.Error(w, "missing pool_id or user_id", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		log.Printf("[nbgrader] collect SSH error: %v", err)
		http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer client.Close()

	innerCmd := "cd /home/jovyan/nbgrader && nbgrader collect"
	if assignment != "" {
		innerCmd = fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader collect %s", assignment)
	}
	out, err := runSSHOutput(client, dockerExec(innerCmd))
	if err != nil {
		log.Printf("[nbgrader] collect error: %v", err)
		http.Error(w, "nbgrader collect failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"output": out,
	})
}

// handleNbgraderAutograde runs `nbgrader autograde` on the instructor VM.
// POST /api/nbgrader/autograde?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderAutograde(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" || assignment == "" {
		http.Error(w, "missing pool_id, user_id or assignment", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		log.Printf("[nbgrader] autograde SSH error: %v", err)
		http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer client.Close()

	cmd := dockerExec(fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader autograde %s", assignment))
	out, err := runSSHOutput(client, cmd)
	if err != nil {
		log.Printf("[nbgrader] autograde error: %v", err)
		// Return partial output even on error (nbgrader may exit non-zero but still grade)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "error",
			"output": out + "\n" + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"assignment": assignment,
		"output":     out,
	})
}

// NbgraderGrade represents one student's grade for an assignment.
type NbgraderGrade struct {
	Student  string  `json:"student"`
	Score    float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
	Status   string  `json:"status"` // "graded", "missing", "needs_manual_grade"
}

// handleNbgraderGrades reads the gradebook via `nbgrader export` CSV on the instructor VM.
// GET /api/nbgrader/grades?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderGrades(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" || assignment == "" {
		http.Error(w, "missing pool_id, user_id or assignment", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		log.Printf("[nbgrader] grades SSH error: %v", err)
		http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer client.Close()

	// Export CSV then filter by assignment, parse student/score/max_score columns
	tmpFile := fmt.Sprintf("/tmp/nbgrader_export_%d.csv", time.Now().UnixNano())
	innerGrades := fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader export --to=%s && cat %s; rm -f %s", tmpFile, tmpFile, tmpFile)
	out, err := runSSHOutput(client, dockerExec(innerGrades))
	// Cleanup in background (best effort)
	go func() {
		s, _ := client.NewSession()
		if s != nil {
			s.Run(fmt.Sprintf("rm -f %q", tmpFile))
			s.Close()
		}
	}()

	if err != nil || out == "" {
		// Fallback: try reading gradebook.db via sqlite3
		sqlInner := fmt.Sprintf(
			`sqlite3 /home/jovyan/nbgrader/gradebook.db "SELECT s.name, nb.score, nb.max_score FROM grade nb JOIN student s ON nb.student_id = s.id JOIN notebook n ON nb.notebook_id = n.id JOIN assignment a ON n.assignment_id = a.id WHERE a.name='%s' ORDER BY s.name;" 2>/dev/null || echo ""`,
			strings.ReplaceAll(assignment, "'", "''"),
		)
		out2, err2 := runSSHOutput(client, dockerExec(sqlInner))
		if err2 != nil || out2 == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"grades": []NbgraderGrade{}})
			return
		}
		grades := parseSQLiteGrades(out2)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"grades": grades})
		return
	}

	grades := parseCSVGrades(out, assignment)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"grades": grades})
}

// parseCSVGrades parses nbgrader CSV export (columns: student_id,assignment,score,max_score,needs_manual_grade)
func parseCSVGrades(csv, assignment string) []NbgraderGrade {
	var grades []NbgraderGrade
	lines := strings.Split(csv, "\n")
	if len(lines) < 2 {
		return grades
	}
	// Find column indices from header
	header := strings.Split(lines[0], ",")
	idx := func(name string) int {
		for i, h := range header {
			if strings.TrimSpace(h) == name {
				return i
			}
		}
		return -1
	}
	iStudent := idx("student_id")
	iAssign := idx("assignment")
	iScore := idx("score")
	iMax := idx("max_score")
	iNMG := idx("needs_manual_grade")
	if iStudent < 0 || iScore < 0 {
		return grades
	}

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cols := strings.Split(line, ",")
		if len(cols) <= iStudent {
			continue
		}
		if iAssign >= 0 && len(cols) > iAssign && strings.TrimSpace(cols[iAssign]) != assignment {
			continue
		}
		g := NbgraderGrade{Student: strings.TrimSpace(cols[iStudent])}
		if iScore >= 0 && len(cols) > iScore {
			fmt.Sscanf(cols[iScore], "%f", &g.Score)
		}
		if iMax >= 0 && len(cols) > iMax {
			fmt.Sscanf(cols[iMax], "%f", &g.MaxScore)
		}
		g.Status = "graded"
		if iNMG >= 0 && len(cols) > iNMG && strings.TrimSpace(cols[iNMG]) == "True" {
			g.Status = "needs_manual_grade"
		}
		grades = append(grades, g)
	}
	return grades
}

// handleNbgraderJupyterURL returns the JupyterLab URL for the instructor VM.
// GET /api/nbgrader/jupyter-url?pool_id=X&user_id=Y
func handleNbgraderJupyterURL(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	if poolID == "" || userID == "" {
		http.Error(w, "missing pool_id or user_id", http.StatusBadRequest)
		return
	}

	var server models.Server
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		First(&server).Error; err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"url": ""})
		return
	}

	ip := server.IP_Address
	if server.Networks != nil {
		for _, net := range server.Networks {
			if idx := strings.LastIndex(net, ":"); idx >= 0 {
				ip = net[idx+1:]
				break
			}
		}
	}

	// Get app_port from the pool (defaults to 8888 for JupyterLab)
	var pool models.Serverpool
	port := 8888
	if err := config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&pool).Error; err == nil {
		if pool.AppPort > 0 {
			port = pool.AppPort
		}
	}

	// Return both the direct URL and the proxied URL (via control center → avoids mixed-content)
	directURL := fmt.Sprintf("http://%s:%d", ip, port)
	// Encode @ explicitly since url.PathEscape doesn't encode it but Caddy rejects it in path segments
	encodedUserID := strings.ReplaceAll(url.PathEscape(userID), "@", "%40")
	proxyURL := fmt.Sprintf("/api/jupyter-proxy/%s/%s/", poolID, encodedUserID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url":       proxyURL,
		"directUrl": directURL,
	})
}

// handleNbgraderRelease releases an assignment from the instructor VM to all student VMs.
// POST /api/nbgrader/release?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" || assignment == "" {
		http.Error(w, "missing pool_id, user_id or assignment", http.StatusBadRequest)
		return
	}

	// 1. SSH on instructor VM → nbgrader release
	instrClient, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		// Fallback: try any pool with this name (non-instructor pools can also have JupyterLab)
		instrClient, err = nbgraderSSHClientAny(poolID, userID)
		if err != nil {
			log.Printf("[nbgrader] release SSH error: %v", err)
			http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	defer instrClient.Close()

	releaseInner := fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader release_assignment %s 2>&1 || nbgrader release %s 2>&1", assignment, assignment)
	releaseOut, err := runSSHOutput(instrClient, dockerExec(releaseInner))
	if err != nil {
		log.Printf("[nbgrader] release command error: %v", err)
		// Don't fail — the release dir may still exist from a previous run
	}

	// 2. Read released files from instructor VM
	fileListInner := fmt.Sprintf(`find /home/jovyan/nbgrader/release/%s -type f 2>/dev/null | sed "s|/home/jovyan/nbgrader/release/%s/||" || echo ""`, assignment, assignment)
	fileList, err := runSSHOutput(instrClient, dockerExec(fileListInner))
	if err != nil || fileList == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"output":  releaseOut,
			"message": "No files found in release directory",
		})
		return
	}

	// 3. Get instructor VM IP for SCP
	var instrServer models.Server
	config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&instrServer)

	// 4. Get SSH key
	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = os.Getenv("SSH_KEY_PATH")
	}
	signer, err := sshinject.LoadPrivateKey(keyPath)
	if err != nil {
		http.Error(w, "load SSH key: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Find all student VMs for this pool
	var pool models.Serverpool
	if err := config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&pool).Error; err != nil {
		http.Error(w, "pool not found", http.StatusNotFound)
		return
	}

	var list models.ListStudents
	config.Database.Preload("Students").Where("pool_id = ?", pool.ID).First(&list)

	distributed := 0
	var distErrors []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, student := range list.Students {
		if student.IP == "" {
			continue
		}
		wg.Add(1)
		go func(s models.Student) {
			defer wg.Done()
			cfg := sshinject.SshConfig("vmuser", signer)
			studentClient, err := ssh.Dial("tcp", s.IP+":22", cfg)
			if err != nil {
				mu.Lock()
				distErrors = append(distErrors, fmt.Sprintf("%s: SSH dial failed: %v", s.Name, err))
				mu.Unlock()
				return
			}
			defer studentClient.Close()

			// Create assignment directory
			mkdirCmd := fmt.Sprintf("mkdir -p ~/assignments/%q", assignment)
			if err := sshinject.RunSSHcmd(studentClient, mkdirCmd); err != nil {
				mu.Lock()
				distErrors = append(distErrors, fmt.Sprintf("%s: mkdir failed: %v", s.Name, err))
				mu.Unlock()
				return
			}

			// SCP files from instructor to student via control center (read from instructor, write to student)
			for _, relFile := range strings.Split(strings.TrimSpace(fileList), "\n") {
				relFile = strings.TrimSpace(relFile)
				if relFile == "" {
					continue
				}
				// Read file from instructor VM
				// Read from Docker container via SSH pipe
				catCmd := dockerExec(fmt.Sprintf("cat /home/jovyan/nbgrader/release/%s/%s", assignment, relFile))
				content, readErr := func() ([]byte, error) {
					out, e := runSSHOutput(instrClient, catCmd)
					return []byte(out), e
				}()
				if readErr != nil {
					mu.Lock()
					distErrors = append(distErrors, fmt.Sprintf("%s: read %s failed: %v", s.Name, relFile, readErr))
					mu.Unlock()
					continue
				}
				// Write file to student VM
				destPath := fmt.Sprintf("/home/vmuser/assignments/%s/%s", assignment, relFile)
				if writeErr := scpWriteFile(studentClient, destPath, content); writeErr != nil {
					mu.Lock()
					distErrors = append(distErrors, fmt.Sprintf("%s: write %s failed: %v", s.Name, relFile, writeErr))
					mu.Unlock()
				}
			}

			mu.Lock()
			distributed++
			mu.Unlock()
		}(student)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":      "ok",
		"distributed": distributed,
		"errors":      distErrors,
		"output":      releaseOut,
	})
}

// scpReadFile reads a remote file via SSH/SCP and returns its content.
func scpReadFile(client *ssh.Client, remotePath string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var buf bytes.Buffer
	session.Stdout = &buf
	if err := session.Run(fmt.Sprintf("cat %q", remotePath)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// scpWriteFile writes content to a remote file via SSH.
func scpWriteFile(client *ssh.Client, remotePath string, content []byte) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Ensure parent directory exists
	dir := remotePath[:strings.LastIndex(remotePath, "/")]
	mkSession, _ := client.NewSession()
	if mkSession != nil {
		mkSession.Run(fmt.Sprintf("mkdir -p %q", dir))
		mkSession.Close()
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("cat > %q", remotePath)
	if err := session.Start(cmd); err != nil {
		return err
	}
	if _, err := io.Copy(stdin, bytes.NewReader(content)); err != nil {
		return err
	}
	stdin.Close()
	return session.Wait()
}

// nbgraderSSHClientAny connects to any VM in the pool (not just instructor role).
func nbgraderSSHClientAny(poolID, userID string) (*ssh.Client, error) {
	var server models.Server
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		First(&server).Error; err != nil {
		return nil, fmt.Errorf("no VM found for pool %s/%s: %w", poolID, userID, err)
	}
	if server.IP_Address == "" {
		return nil, fmt.Errorf("VM has no IP address")
	}
	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = os.Getenv("SSH_KEY_PATH")
	}
	signer, err := sshinject.LoadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load SSH key: %w", err)
	}
	cfg := sshinject.SshConfig("vmuser", signer)
	return ssh.Dial("tcp", server.IP_Address+":22", cfg)
}

// handleNbgraderExportCSV exports grades as a downloadable CSV file.
// GET /api/nbgrader/export-csv?pool_id=X&user_id=Y&assignment=Z
func handleNbgraderExportCSV(w http.ResponseWriter, r *http.Request) {
	poolID := r.URL.Query().Get("pool_id")
	userID := r.URL.Query().Get("user_id")
	assignment := r.URL.Query().Get("assignment")
	if poolID == "" || userID == "" {
		http.Error(w, "missing pool_id or user_id", http.StatusBadRequest)
		return
	}

	client, err := nbgraderSSHClient(poolID, userID)
	if err != nil {
		client, err = nbgraderSSHClientAny(poolID, userID)
		if err != nil {
			http.Error(w, "cannot connect to instructor VM: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	defer client.Close()

	tmpFile := fmt.Sprintf("/tmp/nbgrader_export_%d.csv", time.Now().UnixNano())
	exportInner := fmt.Sprintf("cd /home/jovyan/nbgrader && nbgrader export --to=%s 2>/dev/null && cat %s; rm -f %s", tmpFile, tmpFile, tmpFile)
	out, err := runSSHOutput(client, dockerExec(exportInner))
	if err != nil || out == "" {
		// Fallback: build CSV from sqlite
		whereClause := ""
		if assignment != "" {
			whereClause = fmt.Sprintf(" WHERE a.name='%s'", strings.ReplaceAll(assignment, "'", "''"))
		}
		sqlCsvInner := fmt.Sprintf(
			`sqlite3 /home/jovyan/nbgrader/gradebook.db "SELECT s.name, a.name, nb.score, nb.max_score FROM grade nb JOIN student s ON nb.student_id = s.id JOIN notebook n ON nb.notebook_id = n.id JOIN assignment a ON n.assignment_id = a.id%s ORDER BY s.name;" 2>/dev/null || echo ""`,
			whereClause,
		)
		sqlOut, _ := runSSHOutput(client, dockerExec(sqlCsvInner))
		if sqlOut == "" {
			http.Error(w, "no grades available", http.StatusNotFound)
			return
		}
		var sb strings.Builder
		sb.WriteString("student_id,assignment,score,max_score\n")
		for _, line := range strings.Split(sqlOut, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				parts := strings.Split(line, "|")
				if len(parts) == 4 {
					sb.WriteString(strings.Join(parts, ",") + "\n")
				}
			}
		}
		out = sb.String()
	}

	filename := "grades"
	if assignment != "" {
		filename = "grades_" + assignment
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
	w.Write([]byte(out))
}

// parseSQLiteGrades parses sqlite3 pipe-separated output: name|score|max_score
func parseSQLiteGrades(out string) []NbgraderGrade {
	var grades []NbgraderGrade
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		g := NbgraderGrade{Student: strings.TrimSpace(parts[0]), Status: "graded"}
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &g.Score)
		fmt.Sscanf(strings.TrimSpace(parts[2]), "%f", &g.MaxScore)
		grades = append(grades, g)
	}
	return grades
}
