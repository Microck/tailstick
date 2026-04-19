// Package gui serves the browser-based setup wizard via a local HTTP server.
// It embeds static assets (HTML, favicon) and exposes REST endpoints for
// preset selection, mode configuration, and lease creation.
package gui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tailstick/tailstick/internal/config"
	"github.com/tailstick/tailstick/internal/model"
)

//go:embed index.html tailstick-favicon.png
var staticFS embed.FS

type Server struct {
	ConfigPath string
	Logf       func(format string, args ...any)
	EnrollFn   func(context.Context, model.RuntimeOptions) (model.LeaseRecord, error)
}

type enrollRequest struct {
	PresetID      string `json:"presetId"`
	Mode          string `json:"mode"`
	Channel       string `json:"channel"`
	Days          int    `json:"days"`
	CustomDays    int    `json:"customDays"`
	Suffix        string `json:"suffix"`
	ExitNode      string `json:"exitNode"`
	AllowExisting bool   `json:"allowExisting"`
	Password      string `json:"password"`
}

type presetSummary struct {
	ID                     string   `json:"id"`
	Description            string   `json:"description"`
	Tags                   []string `json:"tags"`
	AcceptRoutes           bool     `json:"acceptRoutes"`
	AllowExitNodeSelection bool     `json:"allowExitNodeSelection"`
	ApprovedExitNodes      []string `json:"approvedExitNodes"`
}

var validModes = map[string]bool{
	string(model.LeaseModeSession):   true,
	string(model.LeaseModeTimed):     true,
	string(model.LeaseModePermanent): true,
}

var validChannels = map[string]bool{
	string(model.ChannelStable): true,
	string(model.ChannelLatest): true,
}

func Run(ctx context.Context, srv *Server, openBrowser bool, host string, port int) error {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	if port < 0 || port > 65535 {
		return fmt.Errorf("invalid port %d: expected 0-65535", port)
	}

	listenAddr := net.JoinHostPort(host, strconv.Itoa(port))
	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.index)
	mux.HandleFunc("/api/presets", srv.presets)
	mux.HandleFunc("/api/enroll", srv.enroll)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String()
	if srv.Logf != nil {
		srv.Logf("tailstick gui listening on %s", url)
	}
	fmt.Printf("TailStick Web UI: %s\n", url)
	if openBrowser {
		_ = open(url)
	}

	httpSrv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		_ = httpSrv.Shutdown(context.Background())
	}()
	err = httpSrv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) presets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := config.Load(s.ConfigPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	summaries := make([]presetSummary, len(cfg.Presets))
	for i, p := range cfg.Presets {
		summaries[i] = presetSummary{
			ID:                     p.ID,
			Description:            p.Description,
			Tags:                   p.Tags,
			AcceptRoutes:           p.AcceptRoutes,
			AllowExitNodeSelection: p.AllowExitNodeSelection,
			ApprovedExitNodes:      p.ApprovedExitNodes,
		}
	}
	writeJSON(w, map[string]any{"defaultPreset": cfg.DefaultPreset, "presets": summaries})
}

func (s *Server) enroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req enrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if req.Mode != "" && !validModes[req.Mode] {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": fmt.Sprintf("invalid mode %q: must be session, timed, or permanent", req.Mode)})
		return
	}
	if req.Channel != "" && !validChannels[req.Channel] {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": fmt.Sprintf("invalid channel %q: must be stable or latest", req.Channel)})
		return
	}
	if req.Days < 0 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "days must be non-negative"})
		return
	}
	if req.CustomDays < 0 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "customDays must be non-negative"})
		return
	}
	if req.Mode == string(model.LeaseModeTimed) && req.CustomDays == 0 && req.Days == 0 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "timed mode requires days > 0 or customDays > 0"})
		return
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		password = strings.TrimSpace(os.Getenv("TAILSTICK_OPERATOR_PASSWORD"))
	}
	rec, err := s.EnrollFn(r.Context(), model.RuntimeOptions{
		PresetID:      req.PresetID,
		Mode:          model.LeaseMode(req.Mode),
		Channel:       model.Channel(req.Channel),
		Days:          req.Days,
		CustomDays:    req.CustomDays,
		DeviceSuffix:  req.Suffix,
		ExitNode:      req.ExitNode,
		AllowExisting: req.AllowExisting,
		Password:      password,
	})
	if err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	resp := map[string]any{
		"ok":         true,
		"leaseId":    rec.LeaseID,
		"deviceName": rec.DeviceName,
		"mode":       rec.Mode,
	}
	if rec.ExpiresAt != nil {
		resp["expiresAt"] = rec.ExpiresAt
	}
	writeJSON(w, resp)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		b, _ := staticFS.ReadFile("tailstick-favicon.png")
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(b)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, _ := staticFS.ReadFile("index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}

func writeJSON(w http.ResponseWriter, data any) {
	writeJSONCode(w, http.StatusOK, data)
}

func writeJSONCode(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
