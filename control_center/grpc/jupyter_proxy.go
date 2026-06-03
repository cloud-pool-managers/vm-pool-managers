package grpc

import (
	"bytes"
	"control_center/config"
	"control_center/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

func decodePathSegment(s string) string {
	if d, err := url.PathUnescape(s); err == nil {
		return d
	}
	return s
}

// rewriteJupyterHTML prefixes the absolute URLs that JupyterLab/formgrader emit
// in their bootstrap HTML so that, behind a path-prefix proxy, the browser sends
// every follow-up request (assets, API, websockets) back through the proxy.
// prefix ends with "/", e.g. "/api/jupyter-proxy/papy/admin%40x.edu/".
func rewriteJupyterHTML(html, prefix string) string {
	trimmed := strings.TrimSuffix(prefix, "/")
	// PageConfig baseUrl: "/" → prefix
	html = strings.ReplaceAll(html, `"baseUrl": "/"`, `"baseUrl": "`+prefix+`"`)
	html = strings.ReplaceAll(html, `"baseUrl":"/"`, `"baseUrl":"`+prefix+`"`)
	// Absolute asset/app paths (quote-anchored so we never touch substrings like
	// "/static/lab" when prefixing "/lab"). Order matters: static before lab.
	html = strings.ReplaceAll(html, `"/static/`, `"`+trimmed+`/static/`)
	html = strings.ReplaceAll(html, `"/lab`, `"`+trimmed+`/lab`)
	html = strings.ReplaceAll(html, `"/formgrader`, `"`+trimmed+`/formgrader`)
	return html
}

// handleJupyterProxy proxies all requests (HTTP + WebSocket) to a JupyterLab VM.
// URL format: /api/jupyter-proxy/{pool_id}/{user_id}/{...rest}
//
// It solves two problems at once:
//   - mixed content: the browser talks HTTPS (Caddy → control center), which
//     forwards HTTP to the private VM, so JupyterLab can be embedded in an iframe.
//   - path-prefix proxying WITHOUT changing JupyterLab's server base_url (which
//     would break direct access used by the student portal): we strip the prefix
//     before forwarding (Jupyter stays at root) and rewrite the bootstrap HTML's
//     absolute URLs to carry the prefix.
func handleJupyterProxy(w http.ResponseWriter, r *http.Request) {
	// Use the escaped path so the encoded user id (e.g. %40) is preserved and the
	// rewritten prefix matches exactly what the browser will request next.
	escaped := strings.TrimPrefix(r.URL.EscapedPath(), "/api/jupyter-proxy/")
	parts := strings.SplitN(escaped, "/", 3)
	if len(parts) < 2 {
		http.Error(w, "usage: /api/jupyter-proxy/{pool_id}/{user_id}/...", http.StatusBadRequest)
		return
	}
	poolID := decodePathSegment(parts[0])
	userID := decodePathSegment(parts[1])
	prefix := "/api/jupyter-proxy/" + parts[0] + "/" + parts[1] + "/"
	rest := "/"
	if len(parts) == 3 {
		rest += decodePathSegment(parts[2])
	}

	var server models.Server
	if err := config.Database.
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		First(&server).Error; err != nil {
		http.Error(w, "VM not found for pool "+poolID, http.StatusNotFound)
		return
	}
	if server.IP_Address == "" {
		http.Error(w, "VM has no IP address yet", http.StatusServiceUnavailable)
		return
	}

	var pool models.Serverpool
	port := 8888
	if err := config.Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&pool).Error; err == nil && pool.AppPort > 0 {
		port = pool.AppPort
	}

	targetBase, _ := url.Parse(fmt.Sprintf("http://%s:%d", server.IP_Address, port))

	trimmed := strings.TrimSuffix(prefix, "/")
	proxy := httputil.NewSingleHostReverseProxy(targetBase)
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Allow iframe embedding.
		resp.Header.Del("X-Frame-Options")
		resp.Header.Del("Content-Security-Policy")
		// Redirects (e.g. /formgrader → /formgrader/manage_assignments) come back
		// root-absolute without the prefix — re-prefix them so the browser stays
		// inside the proxy path.
		if loc := resp.Header.Get("Location"); strings.HasPrefix(loc, "/") && !strings.HasPrefix(loc, prefix) {
			resp.Header.Set("Location", trimmed+loc)
		}
		// Rewrite absolute URLs only in HTML bootstrap pages.
		if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}
		nb := []byte(rewriteJupyterHTML(string(body), prefix))
		resp.Body = io.NopCloser(bytes.NewReader(nb))
		resp.ContentLength = int64(len(nb))
		resp.Header.Set("Content-Length", strconv.Itoa(len(nb)))
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[jupyter-proxy] %s/%s → %s error: %v", poolID, userID, targetBase.Host, err)
		http.Error(w, "JupyterLab unreachable: "+err.Error(), http.StatusBadGateway)
	}

	// Strip the proxy prefix: Jupyter runs at root, so direct access (student
	// portal) keeps working while the iframe goes through the prefix.
	r2 := r.Clone(r.Context())
	r2.URL.Scheme = targetBase.Scheme
	r2.URL.Host = targetBase.Host
	r2.URL.Path = rest
	r2.URL.RawPath = ""
	r2.Host = targetBase.Host
	// Ask for uncompressed HTML so ModifyResponse can rewrite it.
	r2.Header.Set("Accept-Encoding", "identity")
	// JupyterLab checks Origin — set it to match the target so it accepts the request.
	r2.Header.Set("Origin", targetBase.String())
	r2.Header.Set("X-Forwarded-Proto", "http")

	proxy.ServeHTTP(w, r2)
}
