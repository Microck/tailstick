package gui

import (
	"context"
	_ "embed"
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
	mux.HandleFunc("/favicon.ico", srv.favicon)
	mux.HandleFunc("/favicon.png", srv.favicon)
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
	cfg, err := config.Load(s.ConfigPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"defaultPreset": cfg.DefaultPreset, "presets": cfg.Presets})
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
	writeJSON(w, map[string]any{
		"ok":         true,
		"leaseId":    rec.LeaseID,
		"deviceName": rec.DeviceName,
		"mode":       rec.Mode,
		"expiresAt":  rec.ExpiresAt,
	})
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) favicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(faviconPNG)
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

//go:embed tailstick-favicon.png
var faviconPNG []byte

var indexHTML = strings.TrimSpace(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <link rel="icon" type="image/png" href="/favicon.png" />
  <title>TailStick</title>
  <style>
    :root {
      --bg: #f8fafc;
      --ink: #1f2937;
      --muted: #6b7280;
      --accent: #0f766e;
      --line: #d1d5db;
      --card: #ffffff;
    }
    body { margin: 0; font-family: "Segoe UI", "Inter", sans-serif; background: radial-gradient(circle at top right, #ddfbf7, #f8fafc 45%); color: var(--ink); }
    main { max-width: 860px; margin: 36px auto; padding: 0 20px; }
    .card { background: var(--card); border: 1px solid var(--line); border-radius: 14px; padding: 22px; box-shadow: 0 8px 28px rgba(15, 118, 110, 0.08); }
    h1 { margin: 0 0 6px; letter-spacing: -0.02em; }
    p { margin: 0 0 20px; color: var(--muted); }
    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
    label { display: grid; gap: 6px; font-size: 14px; }
    input, select { border: 1px solid var(--line); border-radius: 10px; padding: 10px 12px; font-size: 14px; }
    .row { margin-top: 14px; }
    button { margin-top: 16px; border: 0; border-radius: 10px; padding: 11px 16px; background: var(--accent); color: #fff; font-weight: 600; cursor: pointer; }
    pre { margin-top: 18px; background: #0b1020; color: #e2e8f0; border-radius: 10px; padding: 12px; overflow: auto; }
    @media (max-width: 780px) { .grid { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <main>
    <div class="card">
      <h1>TailStick Launcher</h1>
      <p>Create a Tailscale lease from USB presets.</p>
      <div class="grid">
        <label>Preset<select id="preset"></select></label>
        <label>Mode<select id="mode"><option value="session">Session</option><option value="timed">Timed</option><option value="permanent">Permanent</option></select></label>
        <label>Channel<select id="channel"><option value="stable">Stable</option><option value="latest">Latest</option></select></label>
        <label>Days<select id="days"><option value="1">1</option><option value="3" selected>3</option><option value="7">7</option></select></label>
        <label><input id="useCustomDays" type="checkbox" /> Enable custom days (advanced)</label>
        <label>Custom Days<input id="customDays" type="number" min="1" max="30" value="0" disabled /></label>
        <label>Device Suffix<input id="suffix" placeholder="optional-label" /></label>
        <label>Exit Node<input id="exitNode" placeholder="approved-exit-node" /></label>
        <label>Operator Password (optional)<input id="password" type="password" /></label>
      </div>
      <div class="row">
        <label><input id="allowExisting" type="checkbox" /> Allow existing Tailscale install</label>
      </div>
      <button id="run">Create Lease</button>
      <pre id="out">Ready.</pre>
    </div>
  </main>
  <script>
    const out = document.getElementById("out");
    function log(v) { out.textContent = typeof v === "string" ? v : JSON.stringify(v, null, 2); }

    async function loadPresets() {
      const r = await fetch("/api/presets");
      const data = await r.json();
      const el = document.getElementById("preset");
      el.innerHTML = "";
      for (const p of data.presets || []) {
        const op = document.createElement("option");
        op.value = p.id;
        op.textContent = p.id + " - " + (p.description || "");
        if (data.defaultPreset && p.id === data.defaultPreset) op.selected = true;
        el.appendChild(op);
      }
    }

    document.getElementById("run").addEventListener("click", async () => {
      const useCustom = document.getElementById("useCustomDays").checked;
      const body = {
        presetId: document.getElementById("preset").value,
        mode: document.getElementById("mode").value,
        channel: document.getElementById("channel").value,
        days: Number(document.getElementById("days").value),
        customDays: useCustom ? Number(document.getElementById("customDays").value) : 0,
        suffix: document.getElementById("suffix").value,
        exitNode: document.getElementById("exitNode").value,
        allowExisting: document.getElementById("allowExisting").checked,
        password: document.getElementById("password").value
      };
      const r = await fetch("/api/enroll", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) });
      const data = await r.json();
      log(data);
    });
    document.getElementById("useCustomDays").addEventListener("change", (e) => {
      document.getElementById("customDays").disabled = !e.target.checked;
    });
    loadPresets().catch(err => log(String(err)));
  </script>
</body>
</html>`)
