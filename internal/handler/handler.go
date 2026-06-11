package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/CaptDany/kestrel/internal/db"
	"github.com/CaptDany/kestrel/internal/engine"
	"github.com/CaptDany/kestrel/internal/extractor"
	"github.com/CaptDany/kestrel/internal/notifier"
	"github.com/CaptDany/kestrel/internal/tracker"
)

type Handler struct {
	tpl     *template.Template
	notify  *notifier.Engine
	tracker *tracker.Tracker
}

func NewHandler(tpl *template.Template) *Handler {
	return &Handler{tpl: tpl}
}

func (h *Handler) SetNotifierEngine(e *notifier.Engine) {
	h.notify = e
}

func (h *Handler) SetTracker(t *tracker.Tracker) {
	h.tracker = t
}

type pageData struct {
	Title string
	Data  interface{}
	Error string
}

func (h *Handler) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render error: %v", err)
		http.Error(w, "Internal error", 500)
	}
}

func jsonResp(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, status int, msg string) {
	jsonResp(w, status, map[string]string{"error": msg})
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// ─── Dashboard ──────────────────────────────────────────────────────────────

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	plan, _ := db.GetPlan()
	settings, _ := db.GetAllSettings()
	allItems, _ := db.GetItems("")

	pendingCount := 0
	purchasedCount := 0
	savingCount := 0
	for _, item := range allItems {
		switch item.Status {
		case "pending":
			pendingCount++
		case "purchased":
			purchasedCount++
		case "saving":
			savingCount++
		}
	}

	nextPurchase := ""
	totalPlanned := 0.0
	for _, p := range plan {
		if p.Status == "planned" || p.Status == "" {
			if nextPurchase == "" || p.ScheduledDate < nextPurchase {
				nextPurchase = p.ScheduledDate
			}
			if p.AmountAllocated != nil {
				totalPlanned += *p.AmountAllocated
			}
		}
	}

	priceDropCount, _ := db.GetPriceDropCountToday()

	h.render(w, "dashboard.html", pageData{
		Title: "Dashboard",
		Data: map[string]interface{}{
			"pendingCount":   pendingCount,
			"purchasedCount": purchasedCount,
			"savingCount":    savingCount,
			"priceDropCount": priceDropCount,
			"nextPurchase":   nextPurchase,
			"totalPlanned":   totalPlanned,
			"plan":           plan,
			"settings":       settings,
		},
	})
}

// ─── Items ──────────────────────────────────────────────────────────────────

func (h *Handler) ItemsPage(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	items, _ := db.GetItems(status)
	h.render(w, "items.html", pageData{
		Title: "Items",
		Data:  items,
	})
}

func (h *Handler) ItemNew(w http.ResponseWriter, r *http.Request) {
	h.render(w, "item_form.html", pageData{
		Title: "Add Item",
		Data:  nil,
	})
}

func (h *Handler) ItemEdit(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		jsonErr(w, 400, "invalid id")
		return
	}
	item, err := db.GetItem(id)
	if err != nil {
		jsonErr(w, 404, "item not found")
		return
	}
	h.render(w, "item_form.html", pageData{
		Title: "Edit Item",
		Data:  item,
	})
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var item db.Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		jsonErr(w, 400, "invalid json")
		return
	}
	if item.Status == "" {
		item.Status = "pending"
	}
	id, err := db.CreateItem(&item)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	created, _ := db.GetItem(id)
	jsonResp(w, 201, created)
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		jsonErr(w, 400, "invalid id")
		return
	}
	var item db.Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		jsonErr(w, 400, "invalid json")
		return
	}
	item.ID = id
	if err := db.UpdateItem(&item); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	updated, _ := db.GetItem(id)
	jsonResp(w, 200, updated)
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		jsonErr(w, 400, "invalid id")
		return
	}
	if err := db.DeleteItem(id); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResp(w, 200, map[string]string{"status": "deleted"})
}

func (h *Handler) PurchaseItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		jsonErr(w, 400, "invalid id")
		return
	}
	if err := db.MarkItemPurchased(id); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResp(w, 200, map[string]string{"status": "purchased"})
}

// ─── Scrape ─────────────────────────────────────────────────────────────────

func (h *Handler) ScrapeURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		jsonErr(w, 400, "url required")
		return
	}

	u, err := url.Parse(req.URL)
	if err != nil {
		jsonErr(w, 400, "invalid url")
		return
	}

	parser := extractor.GetParser(u)
	result, err := parser.Extract(req.URL)
	if err != nil {
		jsonErr(w, 422, "extraction failed: "+err.Error())
		return
	}

	jsonResp(w, 200, result)
}

// ─── Plan ───────────────────────────────────────────────────────────────────

func (h *Handler) GeneratePlan(w http.ResponseWriter, r *http.Request) {
	settings, _ := db.GetAllSettings()
	paydays, _ := db.GetPaydays()

	planner := engine.NewPlanner(settings, paydays)
	if err := planner.Generate(); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}

	plan, _ := db.GetPlan()
	jsonResp(w, 200, plan)
}

func (h *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
	plan, _ := db.GetPlan()
	jsonResp(w, 200, plan)
}

// ─── Schedule Page ──────────────────────────────────────────────────────────

func (h *Handler) SchedulePage(w http.ResponseWriter, r *http.Request) {
	plan, _ := db.GetPlan()
	items, _ := db.GetItems("")
	paydays, _ := db.GetPaydays()
	settings, _ := db.GetAllSettings()

	h.render(w, "schedule.html", pageData{
		Title: "Schedule",
		Data: map[string]interface{}{
			"plan":     plan,
			"items":    items,
			"paydays":  paydays,
			"settings": settings,
		},
	})
}

// ─── Settings Page ──────────────────────────────────────────────────────────

func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	settings, _ := db.GetAllSettings()
	paydays, _ := db.GetPaydays()
	budgetEntries, _ := db.GetBudgetEntries()

	h.render(w, "settings.html", pageData{
		Title: "Settings",
		Data: map[string]interface{}{
			"settings":      settings,
			"paydays":       paydays,
			"budgetEntries": budgetEntries,
		},
	})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, 400, "invalid json")
		return
	}
	for k, v := range body {
		if err := db.SetSetting(k, v); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
	}
	if h.notify != nil {
		h.notify.RefreshConfig()
	}
	if h.tracker != nil {
		h.tracker.RefreshConfig()
	}
	jsonResp(w, 200, map[string]string{"status": "saved"})
}

func (h *Handler) SendTestNotification(w http.ResponseWriter, r *http.Request) {
	if h.notify == nil {
		jsonErr(w, 500, "notification engine not initialized")
		return
	}
	var req struct {
		Channel string `json:"channel"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.notify.SendTest(req.Channel); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResp(w, 200, map[string]string{"status": "test sent"})
}

// ─── Paydays ────────────────────────────────────────────────────────────────

func (h *Handler) CreatePayday(w http.ResponseWriter, r *http.Request) {
	var p db.Payday
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, 400, "invalid json")
		return
	}
	p.Active = 1
	id, err := db.CreatePayday(&p)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResp(w, 201, map[string]int64{"id": id})
}

func (h *Handler) UpdatePayday(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		jsonErr(w, 400, "invalid id")
		return
	}
	var p db.Payday
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonErr(w, 400, "invalid json")
		return
	}
	p.ID = id
	if err := db.UpdatePayday(&p); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResp(w, 200, map[string]string{"status": "updated"})
}

func (h *Handler) DeletePayday(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		jsonErr(w, 400, "invalid id")
		return
	}
	if err := db.DeletePayday(id); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResp(w, 200, map[string]string{"status": "deleted"})
}

// ─── Budget Entries ─────────────────────────────────────────────────────────

func (h *Handler) GetBudgetEntries(w http.ResponseWriter, r *http.Request) {
	entries, _ := db.GetBudgetEntries()
	jsonResp(w, 200, entries)
}

func (h *Handler) CreateBudgetEntry(w http.ResponseWriter, r *http.Request) {
	var e db.BudgetEntry
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		jsonErr(w, 400, "invalid json")
		return
	}
	id, err := db.CreateBudgetEntry(&e)
	if err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResp(w, 201, map[string]int64{"id": id})
}

func (h *Handler) DeleteBudgetEntry(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		jsonErr(w, 400, "invalid id")
		return
	}
	if err := db.DeleteBudgetEntry(id); err != nil {
		jsonErr(w, 500, err.Error())
		return
	}
	jsonResp(w, 200, map[string]string{"status": "deleted"})
}
