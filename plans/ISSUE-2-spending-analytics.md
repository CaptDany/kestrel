# Plan: Spending Analytics Dashboard (Issue #2)

## Goal

Add visual analytics to the dashboard: category donut chart, monthly spend trend bars, savings progress bars — all rendered with pure CSS/SVG, zero external dependencies.

---

## 1. Data Structures (`internal/db/models.go`)

```go
type CategoryBreakdown struct {
    Category string  `json:"category"`
    Total    float64 `json:"total"`
    Count    int     `json:"count"`
    Color    string  `json:"color"` // assigned by handler for conic-gradient segments
}

type MonthlyTrend struct {
    Month   string  `json:"month"`   // "2026-01"
    Label   string  `json:"label"`   // "Jan 2026"
    Planned float64 `json:"planned"`
    Actual  float64 `json:"actual"`  // 0 until purchase history is tracked (stub for now)
}

type SavingProgress struct {
    ItemID       int64   `json:"item_id"`
    ItemTitle    string  `json:"item_title"`
    TargetPrice  float64 `json:"target_price"`
    Accumulated  float64 `json:"accumulated"`
    Percent      float64 `json:"percent"` // 0-100
}
```

Also add a top-level `DashboardAnalytics` struct:

```go
type DashboardAnalytics struct {
    CategoryBreakdown []CategoryBreakdown `json:"category_breakdown"`
    MonthlyTrend      []MonthlyTrend      `json:"monthly_trend"`
    SavingProgress    []SavingProgress    `json:"saving_progress"`
}
```

---

## 2. DB Queries (`internal/db/queries.go`)

### `GetCategoryBreakdown() ([]CategoryBreakdown, error)`

```sql
SELECT
    COALESCE(NULLIF(category, ''), 'Uncategorized') AS category,
    COALESCE(SUM(price), 0) AS total,
    COUNT(*) AS count
FROM items
WHERE status != 'purchased'
GROUP BY category
ORDER BY total DESC
```

### `GetMonthlyTrend() ([]MonthlyTrend, error)`

```sql
SELECT
    strftime('%Y-%m', scheduled_date) AS month,
    SUM(COALESCE(amount_allocated, 0)) AS planned
FROM purchase_plan
WHERE status IN ('planned', '')
GROUP BY strftime('%Y-%m', scheduled_date)
ORDER BY month ASC
```

For now `Actual` will be `0` — a future issue (#7 Purchase History) will populate it from `purchased_at` data.

### `GetSavingProgress() ([]SavingProgress, error)`

```sql
SELECT
    is.item_id,
    i.title AS item_title,
    COALESCE(i.price, 0) AS target_price,
    is.accumulated
FROM item_savings is
JOIN items i ON i.id = is.item_id
WHERE i.status = 'saving'
ORDER BY (is.accumulated / NULLIF(i.price, 0)) ASC
```

---

## 3. Dashboard Colors

Assign consistent colors per category on the Go side so the donut chart is reproducible. Use the Material 60/30/10 palette:

```go
var categoryColors = []string{
    "#375BD2", // primary — Electronics
    "#FFB690", // tertiary — Home
    "#A84800", // tertiary-container — Clothing
    "#B7C4FF", // primary-fixed-dim — Food
    "#6B7280", // neutral — Other
    "#F59E0B", // amber — default spare
}
```

In the handler, cycle through these for each `CategoryBreakdown`.

---

## 4. Handler Changes (`internal/handler/handler.go`)

In `Dashboard()`:

```go
categoryBreakdown, _ := db.GetCategoryBreakdown()
// assign colors
for i := range categoryBreakdown {
    categoryBreakdown[i].Color = categoryColors[i%len(categoryColors)]
}

monthlyTrend, _ := db.GetMonthlyTrend()
savingProgress, _ := db.GetSavingProgress()
```

Add to the Data map:

```go
"categoryBreakdown": categoryBreakdown,
"monthlyTrend":      monthlyTrend,
"savingProgress":    savingProgress,
```

### Caching (optional for v1)

For v1 the DB queries are fast enough (SQLite with tiny datasets). Skip caching for now — add only if profiling shows >200ms render times. Cache would be a `sync.Map` with 5-minute TTL keys invalidated on plan generate / item mutation.

---

## 5. Template Changes (`ui/templates/dashboard.html`)

### 5a. Category Donut Chart

Insert before the Purchase Plan section (after the stats row). Use CSS `conic-gradient` on a `<div>`:

```
<!-- Category Donut -->
<div class="glass-card p-lg rounded-xl">
  <h3 class="font-headline-md text-headline-md text-on-surface mb-2">Spending by Category</h3>
  <div class="flex items-center gap-6">
    <!-- Donut -->
    <div class="w-32 h-32 rounded-full shrink-0"
         style="background: conic-gradient(
           #COLOR1 0deg DEG1deg,
           #COLOR2 DEG1deg DEG2deg,
           ...
         )">
    </div>
    <!-- Legend -->
    <div class="flex flex-col gap-2">
      {{range .categoryBreakdown}}
      <div class="flex items-center gap-2">
        <div class="w-3 h-3 rounded-full" style="background: {{.Color}}"></div>
        <span class="text-sm text-on-surface-variant">{{.Category}}</span>
        <span class="text-sm text-on-surface font-semibold">{{printf "$%.0f" .Total}}</span>
      </div>
      {{end}}
    </div>
  </div>
</div>
```

Calculation in the template (or pre-calculated in the handler): for each category, `angle = (total / grandTotal) * 360`. Store as cumulative degrees in the struct.

To keep templates clean, pre-calculate `ConicGradient` as a CSS string and `Segments` as a slice of `{DegStart, DegEnd, Color}` on the Go side, then render with a simple range loop.

### 5b. Monthly Trend Bar Chart

A vertical bar chart using div-based proportional heights:

```
<!-- Monthly Trend -->
<div class="glass-card p-lg rounded-xl">
  <h3 class="font-headline-md text-headline-md text-on-surface mb-2">Monthly Spend</h3>
  <div class="flex items-end gap-3 h-40">
    {{range .monthlyTrend}}
    <div class="flex-1 flex flex-col items-center gap-1 h-full justify-end">
      <div class="w-full bg-primary-container rounded-t-lg transition-all"
           style="height: {{.PctOfMax}}%"></div>
      <span class="text-[10px] text-on-surface-variant">{{.Label}}</span>
    </div>
    {{end}}
  </div>
</div>
```

Pre-calculate `PctOfMax` on the Go side: `trend[i].PctOfMax = (trend[i].Planned / maxPlanned) * 100`.

### 5c. Savings Progress Bars

If there are saving items, show a list with progress bars:

```
<!-- Savings Progress -->
<div class="glass-card p-lg rounded-xl">
  <h3 class="font-headline-md text-headline-md text-on-surface mb-2">Savings Progress</h3>
  <div class="flex flex-col gap-3">
    {{range .savingProgress}}
    <div>
      <div class="flex justify-between text-sm mb-1">
        <span class="text-on-surface truncate">{{.ItemTitle}}</span>
        <span class="text-on-surface-variant">{{printf "$%.0f" .Accumulated}} / {{printf "$%.0f" .TargetPrice}}</span>
      </div>
      <div class="w-full h-2 bg-surface-variant rounded-full overflow-hidden">
        <div class="h-full bg-primary rounded-full transition-all" style="width: {{.Percent}}%"></div>
      </div>
    </div>
    {{end}}
  </div>
</div>
```

---

## 6. Layout

Position the three new analytics cards in a responsive grid after the stats row, before the existing Purchase Plan + Budget sidebar:

```
<section class="grid grid-cols-1 lg:grid-cols-12 gap-xl items-start">
  <!-- Analytics row (spans full width on new row) -->
  <div class="lg:col-span-12 grid grid-cols-1 md:grid-cols-3 gap-md">
    <div>Donut Chart</div>
    <div>Monthly Trend</div>
    <div>Savings Progress</div>
  </div>

  <!-- Existing: Purchase Plan (lg:col-span-8) + Budget (lg:col-span-4) -->
  <section class="lg:col-span-8">...</section>
  <section class="lg:col-span-4">...</section>
</section>
```

On mobile (md:), analytics stack vertically. On desktop, all three cards sit side-by-side.

---

## 7. Edge Cases

| Case | Handling |
|------|----------|
| No items or plan | All three sections render empty state (subtle "No data" text) |
| Single category | Full circle with one color |
| Zero total spend | Donut renders as full ring in neutral gray |
| Savings with no items in saving mode | Section hidden entirely |
| Monthly trend with one month shown | Single bar fills full width |
| Very long category names | Truncate with CSS `text-overflow: ellipsis` |

---

## 8. Implementation Order

1. Add structs to `internal/db/models.go`
2. Add three aggregation queries to `internal/db/queries.go`
3. Update `Dashboard()` handler to run queries and pass data
4. Rewrite the analytics section in `ui/templates/dashboard.html`
5. Verify rendering with `go build ./...`
6. Run `go test ./...`

---

## 9. Files Modified

| File | Change |
|------|--------|
| `internal/db/models.go` | Add `CategoryBreakdown`, `MonthlyTrend`, `SavingProgress` structs |
| `internal/db/queries.go` | Add `GetCategoryBreakdown()`, `GetMonthlyTrend()`, `GetSavingProgress()` |
| `internal/handler/handler.go` | Update `Dashboard()` to query analytics and assign colors |
| `ui/templates/dashboard.html` | Add donut chart, bar chart, progress bars |

---

## 10. Acceptance Criteria

- [ ] Category donut chart renders with correct proportional segments and colors
- [ ] Monthly trend bar chart shows planned spend per month with proportional heights
- [ ] Savings progress bars show accumulated vs. target for each saving item
- [ ] Empty states render gracefully when there is no data
- [ ] `go build ./...` and `go test ./...` pass
