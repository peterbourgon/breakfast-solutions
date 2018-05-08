package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type api struct {
	pre  preprocessor
	repo repository
	post postprocessor
	*mux.Router
}

func newAPI(pre preprocessor, repo repository, post postprocessor, imagedir string) *api {
	a := &api{
		pre:  pre,
		repo: repo,
		post: post,
	}
	r := mux.NewRouter()
	{
		r.StrictSlash(true)
		r.Methods("GET").Path("/").HandlerFunc(a.handleRoot)
		r.Methods("GET").Path("/breakfasts/{id:[0-9]+}").HandlerFunc(a.handleGetBreakfast)
		r.Methods("GET").PathPrefix("/images").Handler(http.StripPrefix("/images", http.FileServer(http.Dir(imagedir))))
		r.Methods("GET").Path("/admin").HandlerFunc(a.handleAdmin)
	}
	a.Router = r
	return a
}

func (a *api) handleRoot(w http.ResponseWriter, r *http.Request) {
	var (
		username = getUsername(r)
		region   = getRegion(r)
	)

	a.pre(r.Context(), region)

	b, err := a.repo.getRandomBreakfast(r.Context(), username)

	a.post(r.Context(), username, err == nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Cache-Control", "private") // don't cache, it's random!
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeHTML(w, b)
}

func (a *api) handleGetBreakfast(w http.ResponseWriter, r *http.Request) {
	var (
		username = getUsername(r)
		region   = getRegion(r)
		id, _    = strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
	)

	a.pre(r.Context(), region)

	b, err := a.repo.getBreakfast(r.Context(), username, id)

	a.post(r.Context(), username, err == nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeHTML(w, b)
}

func (a *api) handleAdmin(w http.ResponseWriter, r *http.Request) {
	code, _ := strconv.Atoi(r.URL.Query().Get("code"))
	if code == 0 {
		code = http.StatusUnauthorized
	}
	http.Error(w, fmt.Sprintf("admin returning %d", code), code)
}

func getUsername(r *http.Request) string {
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "<anonymous>"
	}
	return username
}

func getRegion(r *http.Request) string {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "??"
	}
	return region
}

func writeHTML(w io.Writer, b breakfast) {
	fmt.Fprintf(w, "<html><head><title>Breakfast Solutions</title>\n")
	fmt.Fprintf(w, "<style>body { margin: 2em auto; max-width: 500px; }</style></head>\n")
	fmt.Fprintf(w, "<h1>Breakfast Solutions</h1>\n")
	fmt.Fprintf(w, `<h2>%s</h2>`+"\n", b.Name)
	fmt.Fprintf(w, "<br/>\n")
	fmt.Fprintf(w, `<img src="%s" style="max-width:500px;"/>`+"\n", b.Image)
	fmt.Fprintf(w, "<br/>\n")
	fmt.Fprintf(w, "<br/>\n")
	fmt.Fprintf(w, "%s\n", b.Description)
	fmt.Fprintf(w, "<br/>\n")
	fmt.Fprintf(w, `<a href="/breakfasts/%d">Permalink</a>`+"\n", b.ID)
	fmt.Fprintf(w, "</body></html>\n")
}
