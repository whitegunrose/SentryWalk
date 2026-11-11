package main

// Tap-to-scan from a link on the tag (no app needed).
//
// Each NFC tag holds a URL like
//
//	http://192.168.0.23:8000/scan?tag=04A19F22
//
// Tapping the tag with an unlocked Android phone (or an iPhone XS or newer,
// then tapping the banner) opens it in the browser. The guard's ID is not on
// the tag, because every guard taps the same tags. The guard sets it once on
// /guard and the browser keeps it in a cookie.
//
// Scans saved this way go to the same E_live.csv and scans.jsonl as POST /scans.

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// guardCookie is the cookie that remembers which guard is using this browser.
const guardCookie = "sentrywalk_guard"

// handleGuardForm shows the "set your guard ID" page.
// ?next=/scan?tag=... sends the guard back to the scan after saving.
func (s *Server) handleGuardForm(w http.ResponseWriter, r *http.Request) {
	current := ""
	if c, err := r.Cookie(guardCookie); err == nil {
		current = c.Value
	}
	renderPage(w, http.StatusOK, guardPage, map[string]string{
		"Current": current,
		"Next":    safeNext(r.URL.Query().Get("next")),
	})
}

// handleGuardSave stores the guard ID in a cookie.
func (s *Server) handleGuardSave(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.FormValue("id"))
	next := safeNext(r.FormValue("next"))
	if msg := checkGuardID(id); msg != "" {
		renderPage(w, http.StatusUnprocessableEntity, guardPage, map[string]string{"Error": msg, "Next": next})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     guardCookie,
		Value:    id,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60, // one year
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	log.Printf("[guard] browser set to guard=%s", id)
	if next != "" {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	renderPage(w, http.StatusOK, guardPage, map[string]string{"Current": id, "Saved": "yes"})
}

// handleTap saves a scan from a tag link: GET /scan?tag=04A19F22
func (s *Server) handleTap(w http.ResponseWriter, r *http.Request) {
	tagID := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("tag")))
	if tagID == "" || !hexOnly.MatchString(tagID) {
		renderPage(w, http.StatusUnprocessableEntity, resultPage, map[string]string{
			"Error": "This tag's link has no valid tag ID. It should end in ?tag= followed by hex digits, e.g. ?tag=04A19F22",
		})
		return
	}

	c, err := r.Cookie(guardCookie)
	if err != nil || checkGuardID(c.Value) != "" {
		// First tap on this phone: ask for the guard ID, then come back here.
		http.Redirect(w, r, "/guard?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
		return
	}

	tags, err := loadTags(s.TagsFile)
	if err != nil {
		renderPage(w, http.StatusInternalServerError, resultPage, map[string]string{"Error": "Reading tag list: " + err.Error()})
		return
	}
	var checkpoint *string
	if name, ok := tags[tagID]; ok {
		checkpoint = &name
	}

	// The browser sends the scan the moment the tag is tapped, so the scan
	// happened and arrived at the same time (it is never SYNCED LATE).
	now := toJavaTime(s.Now())
	link := r.URL.RequestURI()
	tagType := "ndef-link"
	rec := ScanRecord{
		ScanID:     "link-" + randomID(),
		Happened:   now,
		Arrived:    now,
		Guard:      c.Value,
		TagID:      tagID,
		Checkpoint: checkpoint,
		NdefURL:    &link,
		TagType:    &tagType,
	}

	s.mu.Lock()
	err = s.appendEvent(rec)
	if err == nil {
		err = s.appendLog(rec)
	}
	if err == nil {
		s.seen[rec.ScanID] = true
	}
	s.mu.Unlock()
	if err != nil {
		renderPage(w, http.StatusInternalServerError, resultPage, map[string]string{"Error": "Saving scan: " + err.Error()})
		return
	}

	name := "UNKNOWN TAG"
	if checkpoint != nil {
		name = *checkpoint
	}
	log.Printf("[tap]  %s  guard=%s  tag=%s  -> %s", rec.Happened, rec.Guard, rec.TagID, name)

	// Redirect to a page that only shows the result, so reloading the
	// browser tab doesn't save the scan a second time.
	q := url.Values{"tag": {tagID}, "guard": {rec.Guard}, "at": {rec.Happened}}
	if checkpoint != nil {
		q.Set("checkpoint", *checkpoint)
	}
	http.Redirect(w, r, "/scan/ok?"+q.Encode(), http.StatusSeeOther)
}

// handleTapResult shows what was saved. It saves nothing itself.
func (s *Server) handleTapResult(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	renderPage(w, http.StatusOK, resultPage, map[string]string{
		"Tag":        q.Get("tag"),
		"Guard":      q.Get("guard"),
		"At":         strings.Replace(q.Get("at"), "T", " ", 1),
		"Checkpoint": q.Get("checkpoint"),
	})
}

// checkGuardID returns an error message, or "" if the ID is fine.
func checkGuardID(id string) string {
	switch {
	case id == "":
		return "Enter your guard ID."
	case len(id) > 32:
		return "Guard ID is too long (32 characters max)."
	case strings.ContainsAny(id, ",\r\n;\""):
		return "Guard ID can't contain commas, semicolons, quotes or line breaks."
	}
	return ""
}

// safeNext only allows returning to a scan link on this server, so /guard
// can't be used to send someone to another website.
func safeNext(next string) string {
	if strings.HasPrefix(next, "/scan?") {
		return next
	}
	return ""
}

func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return time.Now().Format("20060102T150405") + "-" + hex.EncodeToString(b)
}

func renderPage(w http.ResponseWriter, status int, t *template.Template, data map[string]string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := t.Execute(w, data); err != nil {
		log.Printf("rendering page: %v", err)
	}
}

const pageStyle = `<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body{font-family:-apple-system,system-ui,sans-serif;margin:0;padding:24px;background:#f4f5f7;color:#1c1e21}
.card{max-width:420px;margin:0 auto;background:#fff;border-radius:14px;padding:24px;box-shadow:0 1px 4px rgba(0,0,0,.08)}
h1{font-size:22px;margin:0 0 12px}
.big{font-size:44px;font-weight:700;margin:8px 0}
.ok{color:#137333}.warn{color:#b06000}.err{color:#c5221f}
dl{display:grid;grid-template-columns:auto 1fr;gap:6px 14px;margin:16px 0 0}dt{color:#666}dd{margin:0;font-family:ui-monospace,monospace}
input{font-size:20px;padding:10px;width:100%;box-sizing:border-box;border:1px solid #ccc;border-radius:8px;margin:8px 0 14px}
button{font-size:18px;padding:12px;width:100%;border:0;border-radius:8px;background:#1a73e8;color:#fff}
a{color:#1a73e8}
</style>`

var guardPage = template.Must(template.New("guard").Parse(`<!doctype html><html><head><meta charset="utf-8"><title>SentryWalk – Guard</title>` + pageStyle + `</head><body><div class="card">
<h1>SentryWalk guard</h1>
{{if .Saved}}<p class="ok">Saved. This phone will now record scans as <b>{{.Current}}</b>. You can tap tags.</p>
{{else if .Current}}<p>This phone is set to guard <b>{{.Current}}</b>.</p>
{{else}}<p>Enter your guard ID once. This phone remembers it for every tag you tap.</p>{{end}}
{{if .Error}}<p class="err">{{.Error}}</p>{{end}}
<form method="post" action="/guard">
<label for="id">Guard ID</label>
<input id="id" name="id" value="{{.Current}}" autocapitalize="characters" autocomplete="off" required>
<input type="hidden" name="next" value="{{.Next}}">
<button type="submit">{{if .Next}}Save and record scan{{else}}Save{{end}}</button>
</form>
</div></body></html>`))

var resultPage = template.Must(template.New("result").Parse(`<!doctype html><html><head><meta charset="utf-8"><title>SentryWalk – Scan</title>` + pageStyle + `</head><body><div class="card">
{{if .Error}}<h1 class="err">Scan not saved</h1><p>{{.Error}}</p>
{{else if .Checkpoint}}<h1 class="ok">Scan saved ✓</h1><div class="big">Checkpoint {{.Checkpoint}}</div>
{{else}}<h1 class="warn">Saved, but unknown tag</h1><p>This tag isn't in <code>inputs/T_tags.csv</code>. Add <code>{{.Tag}}</code> there to give it a checkpoint.</p>{{end}}
{{if not .Error}}<dl><dt>Guard</dt><dd>{{.Guard}}</dd><dt>Tag</dt><dd>{{.Tag}}</dd><dt>Time</dt><dd>{{.At}}</dd></dl>
<p style="margin-top:20px"><a href="/guard">Not {{.Guard}}? Change guard</a></p>{{end}}
</div></body></html>`))
