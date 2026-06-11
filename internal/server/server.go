package server

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/CaptDany/kestrel/internal/handler"
)

type Server struct {
	Router chi.Router
	Addr   string
	Tpl    *template.Template
}

func New(addr string, tpl *template.Template, staticFS http.FileSystem) *Server {
	s := &Server{Addr: addr, Tpl: tpl}
	s.Router = chi.NewRouter()
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.RealIP)

	h := handler.NewHandler(s.Tpl)

	s.Router.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/static")
		http.FileServer(staticFS).ServeHTTP(w, r)
	})

	s.Router.Get("/", h.Dashboard)
	s.Router.Get("/items", h.ItemsPage)
	s.Router.Get("/items/new", h.ItemNew)
	s.Router.Get("/items/{id}/edit", h.ItemEdit)
	s.Router.Get("/schedule", h.SchedulePage)
	s.Router.Get("/settings", h.SettingsPage)

	s.Router.Post("/api/items", h.CreateItem)
	s.Router.Put("/api/items/{id}", h.UpdateItem)
	s.Router.Delete("/api/items/{id}", h.DeleteItem)
	s.Router.Post("/api/items/{id}/purchase", h.PurchaseItem)
	s.Router.Post("/api/scrape", h.ScrapeURL)
	s.Router.Post("/api/plan/generate", h.GeneratePlan)
	s.Router.Get("/api/plan", h.GetPlan)
	s.Router.Put("/api/settings", h.UpdateSettings)
	s.Router.Post("/api/paydays", h.CreatePayday)
	s.Router.Put("/api/paydays/{id}", h.UpdatePayday)
	s.Router.Delete("/api/paydays/{id}", h.DeletePayday)
	s.Router.Get("/api/budget-entries", h.GetBudgetEntries)
	s.Router.Post("/api/budget-entries", h.CreateBudgetEntry)
	s.Router.Delete("/api/budget-entries/{id}", h.DeleteBudgetEntry)

	return s
}

func (s *Server) Start() error {
	log.Printf("kestrel starting on %s", s.Addr)
	return http.ListenAndServe(s.Addr, s.Router)
}
