package engine

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/CaptDany/kestrel/internal/db"
)

type Planner struct {
	settings map[string]string
	paydays  []db.Payday
}

func NewPlanner(settings map[string]string, paydays []db.Payday) *Planner {
	return &Planner{settings: settings, paydays: paydays}
}

type Cycle struct {
	Date   string
	Label  string
	Budget float64
}

type PlanItem struct {
	db.PurchasePlan
}

func (p *Planner) Generate() error {
	items, err := db.GetPendingAndSavingItems()
	if err != nil {
		return fmt.Errorf("get items: %w", err)
	}
	if len(items) == 0 {
		return nil
	}

	cycles := p.generateCycles(12)
	if len(cycles) == 0 {
		return fmt.Errorf("no paydays configured")
	}

	if err := db.ClearPlan(); err != nil {
		return fmt.Errorf("clear plan: %w", err)
	}

	sortItems(items, p.settings["sort_criteria"])

	purchaseMode := p.settings["purchase_mode"]
	if purchaseMode == "" {
		purchaseMode = "one"
	}

	for _, item := range items {
		savings, _ := db.GetItemSavings(item.ID)
		itemSavings[item.ID] = savings
	}

	for _, cycle := range cycles {
		availableBudget := cycle.Budget
		remaining := availableBudget

		var unscheduled []db.Item
		for _, item := range items {
			if itemAlreadyScheduled(item.ID) {
				continue
			}
			unscheduled = append(unscheduled, item)
		}
		if len(unscheduled) == 0 {
			break
		}

		if purchaseMode == "one" {
			for _, item := range unscheduled {
				saved := itemSavings[item.ID]
				needed := *item.Price - saved
				if needed <= remaining {
					entry := &db.PurchasePlan{
						ItemID:          item.ID,
						ScheduledDate:   cycle.Date,
						BudgetCycle:     cycle.Label,
						Rank:            0,
						AmountAllocated: &needed,
						Status:          "planned",
					}
					planItems[item.ID] = true
					if _, err := db.AddPlanEntry(entry); err != nil {
						return err
					}
					if saved > 0 {
						db.DeleteItemSavings(item.ID)
					}
					if item.Status == "saving" {
						item.Status = "pending"
						db.UpdateItem(&item)
					}
					remaining -= needed
					break
				} else if remaining > 0 {
					newSaved := saved + remaining
					itemSavings[item.ID] = newSaved
					db.UpsertItemSavings(item.ID, newSaved)
					if item.Status != "saving" {
						item.Status = "saving"
						db.UpdateItem(&item)
					}
					remaining = 0
					break
				}
			}
		} else {
			for _, item := range unscheduled {
				saved := itemSavings[item.ID]
				needed := *item.Price - saved
				if needed <= remaining {
					entry := &db.PurchasePlan{
						ItemID:          item.ID,
						ScheduledDate:   cycle.Date,
						BudgetCycle:     cycle.Label,
						Rank:            0,
						AmountAllocated: &needed,
						Status:          "planned",
					}
					planItems[item.ID] = true
					if _, err := db.AddPlanEntry(entry); err != nil {
						return err
					}
					if saved > 0 {
						db.DeleteItemSavings(item.ID)
					}
					if item.Status == "saving" {
						item.Status = "pending"
						db.UpdateItem(&item)
					}
					remaining -= needed
				} else if remaining > 0 {
					newSaved := saved + remaining
					itemSavings[item.ID] = newSaved
					db.UpsertItemSavings(item.ID, newSaved)
					if item.Status != "saving" {
						item.Status = "saving"
						db.UpdateItem(&item)
					}
					remaining = 0
				}
			}
		}
	}

	return nil
}

var (
	itemSavings = make(map[int64]float64)
	planItems   = make(map[int64]bool)
)

func itemAlreadyScheduled(id int64) bool {
	return planItems[id]
}

func (p *Planner) generateCycles(count int) []Cycle {
	cycles := make([]Cycle, 0, count)

	for _, pd := range p.paydays {
		if pd.Active == 0 {
			continue
		}
		next := p.parseNextDate(pd)
		for i := 0; i < count; i++ {
			cycle := Cycle{
				Date:   next.Format("2006-01-02"),
				Label:  fmt.Sprintf("%s - %s", pd.Label, next.Format("2006-01-02")),
				Budget: p.getCycleBudget(next.Format("2006-01-02")),
			}
			cycles = append(cycles, cycle)
			next = p.advancePayday(pd, next)
		}
	}

	sort.Slice(cycles, func(i, j int) bool {
		return cycles[i].Date < cycles[j].Date
	})

	return cycles
}

func (p *Planner) parseNextDate(pd db.Payday) time.Time {
	if pd.NextDate != nil && *pd.NextDate != "" {
		if t, err := time.Parse("2006-01-02", *pd.NextDate); err == nil {
			return t
		}
	}
	return time.Now().AddDate(0, 0, 1)
}

func (p *Planner) advancePayday(pd db.Payday, from time.Time) time.Time {
	switch pd.Frequency {
	case "monthly":
		day := 1
		if pd.DayOfMonth != nil && *pd.DayOfMonth > 0 {
			day = *pd.DayOfMonth
		}
		next := time.Date(from.Year(), from.Month()+1, day, 0, 0, 0, 0, from.Location())
		if pd.DayOfMonth != nil && *pd.DayOfMonth == 99 {
			next = time.Date(from.Year(), from.Month()+2, 0, 0, 0, 0, 0, from.Location())
		}
		return next
	case "biweekly":
		return from.AddDate(0, 0, 14*pd.Interval)
	case "weekly":
		return from.AddDate(0, 0, 7*pd.Interval)
	default:
		return from.AddDate(0, 0, 30)
	}
}

func (p *Planner) getCycleBudget(date string) float64 {
	mode := p.settings["budget_mode"]
	baseAmount := 0.0
	if v := p.settings["budget_amount"]; v != "" {
		fmt.Sscanf(v, "%f", &baseAmount)
	}

	switch mode {
	case "per_payday":
		return baseAmount + p.getExtraBudget(date)
	case "total":
		return p.getTotalRemainingBudget()
	case "flexible":
		return baseAmount + p.getExtraBudget(date)
	default:
		return baseAmount
	}
}

func (p *Planner) getExtraBudget(date string) float64 {
	entries, err := db.GetBudgetEntriesForDate(date)
	if err != nil {
		return 0
	}
	total := 0.0
	for _, e := range entries {
		total += e.Amount
	}
	return total
}

func (p *Planner) getTotalRemainingBudget() float64 {
	totalItems := 0.0
	purchased := 0.0

	allItems, _ := db.GetItems("")
	for _, item := range allItems {
		if item.Price != nil {
			totalItems += *item.Price
			if item.Status == "purchased" {
				purchased += *item.Price
			}
		}
	}

	baseAmount := 0.0
	if v := p.settings["budget_amount"]; v != "" {
		fmt.Sscanf(v, "%f", &baseAmount)
	}

	return math.Max(0, baseAmount-purchased)
}

func sortItems(items []db.Item, criteria string) {
	sort.Slice(items, func(i, j int) bool {
		pi := 0.0
		pj := 0.0
		if items[i].Price != nil {
			pi = *items[i].Price
		}
		if items[j].Price != nil {
			pj = *items[j].Price
		}

		switch criteria {
		case "price_desc":
			return pi > pj
		case "priority":
			piVal := items[i].Priority
			pjVal := items[j].Priority
			if piVal != pjVal {
				return piVal > pjVal
			}
			return items[i].CreatedAt < items[j].CreatedAt
		case "date_added":
			return items[i].CreatedAt < items[j].CreatedAt
		case "desired_date":
			if items[i].DesiredDate != nil && items[j].DesiredDate != nil {
				return *items[i].DesiredDate < *items[j].DesiredDate
			}
			if items[i].DesiredDate != nil {
				return true
			}
			return false
		default:
			return pi < pj
		}
	})
}
