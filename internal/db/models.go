package db

type Item struct {
	ID             int64    `json:"id"`
	URL            string   `json:"url"`
	Title          string   `json:"title"`
	Price          *float64 `json:"price"`
	Currency       string   `json:"currency"`
	Priority       int      `json:"priority"`
	Category       string   `json:"category"`
	Notes          string   `json:"notes"`
	Status         string   `json:"status"`
	DesiredDate    *string  `json:"desired_date"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	PurchasedAt    *string  `json:"purchased_at"`
	PriceConfirmed int      `json:"price_confirmed"`
}

type Payday struct {
	ID         int64  `json:"id"`
	Frequency  string `json:"frequency"`
	DayOfMonth *int   `json:"day_of_month"`
	DayOfWeek  *int   `json:"day_of_week"`
	Interval   int    `json:"interval"`
	Label      string `json:"label"`
	NextDate   *string `json:"next_date"`
	Active     int    `json:"active"`
	CreatedAt  string `json:"created_at"`
}

type BudgetEntry struct {
	ID            int64   `json:"id"`
	Amount        float64 `json:"amount"`
	Label         string  `json:"label"`
	EffectiveDate string  `json:"effective_date"`
	CreatedAt     string  `json:"created_at"`
}

type PurchasePlan struct {
	ID              int64    `json:"id"`
	ItemID          int64    `json:"item_id"`
	ScheduledDate   string   `json:"scheduled_date"`
	PaydayID        *int64   `json:"payday_id"`
	BudgetCycle     string   `json:"budget_cycle"`
	Rank            int      `json:"rank"`
	AmountAllocated *float64 `json:"amount_allocated"`
	Status          string   `json:"status"`
	CreatedAt       string   `json:"created_at"`
	Notes           string   `json:"notes"`
	ItemTitle       string   `json:"item_title"`
	ItemPrice       *float64 `json:"item_price"`
	ItemURL         string   `json:"item_url"`
}

type NotificationLog struct {
	ID        int64  `json:"id"`
	ItemID    *int64 `json:"item_id"`
	Type      string `json:"type"`
	Channel   string `json:"channel"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	SentAt    string `json:"sent_at"`
	Delivered int    `json:"delivered"`
}

type PriceHistory struct {
	ID        int64    `json:"id"`
	ItemID    int64    `json:"item_id"`
	Price     *float64 `json:"price"`
	Currency  string   `json:"currency"`
	ScrapedAt string   `json:"scraped_at"`
}

var defaultSettings = map[string]string{
	"budget_mode":      "per_payday",
	"budget_amount":    "0",
	"sort_criteria":    "price_asc",
	"purchase_mode":    "as_many_as_possible",
	"currency":         "USD",
	"extractor_mode":   "http",
	"scraper_url":      "",
	"playwright_enabled": "false",
	"notify_enabled":       "false",
	"notify_email_enabled": "false",
	"notify_smtp_host":     "",
	"notify_smtp_port":     "587",
	"notify_smtp_user":     "",
	"notify_smtp_pass":     "",
	"notify_smtp_from":     "",
	"notify_smtp_to":       "",
	"notify_push_enabled":  "false",
	"notify_quiet_start":   "22:00",
	"notify_quiet_end":     "08:00",
	"tracker_enabled":             "false",
	"tracker_interval_seconds":    "3600",
	"tracker_drop_threshold_pct":  "5",
	"tracker_drop_threshold_abs":  "0",
}
