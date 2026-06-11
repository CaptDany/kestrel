package db

import (
	"database/sql"
	"fmt"
)

// ─── Items ──────────────────────────────────────────────────────────────────

func GetItems(status string) ([]Item, error) {
	query := "SELECT id, url, title, price, currency, priority, category, notes, status, desired_date, created_at, updated_at, purchased_at, price_confirmed FROM items"
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	return scanItems(query, args...)
}

func GetItem(id int64) (*Item, error) {
	items, err := scanItems("SELECT id, url, title, price, currency, priority, category, notes, status, desired_date, created_at, updated_at, purchased_at, price_confirmed FROM items WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("item not found")
	}
	return &items[0], nil
}

func CreateItem(item *Item) (int64, error) {
	res, err := DB.Exec(
		`INSERT INTO items (url, title, price, currency, priority, category, notes, status, desired_date, price_confirmed)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.URL, item.Title, item.Price, item.Currency, item.Priority,
		item.Category, item.Notes, item.Status, item.DesiredDate, item.PriceConfirmed,
	)
	if err != nil {
		return 0, fmt.Errorf("create item: %w", err)
	}
	return res.LastInsertId()
}

func UpdateItem(item *Item) error {
	_, err := DB.Exec(
		`UPDATE items SET url=?, title=?, price=?, currency=?, priority=?, category=?,
		 notes=?, status=?, desired_date=?, price_confirmed=?, updated_at=datetime('now')
		 WHERE id=?`,
		item.URL, item.Title, item.Price, item.Currency, item.Priority,
		item.Category, item.Notes, item.Status, item.DesiredDate, item.PriceConfirmed,
		item.ID,
	)
	return err
}

func DeleteItem(id int64) error {
	_, err := DB.Exec("DELETE FROM items WHERE id = ?", id)
	return err
}

func MarkItemPurchased(id int64) error {
	_, err := DB.Exec(
		"UPDATE items SET status='purchased', purchased_at=datetime('now'), updated_at=datetime('now') WHERE id=?",
		id,
	)
	return err
}

func GetPendingAndSavingItems() ([]Item, error) {
	return scanItems(
		`SELECT id, url, title, price, currency, priority, category, notes, status, desired_date, created_at, updated_at, purchased_at, price_confirmed
		 FROM items WHERE status IN ('pending', 'saving') ORDER BY
		 CASE WHEN status = 'saving' THEN 0 ELSE 1 END, created_at ASC`,
	)
}

func scanItems(query string, args ...interface{}) ([]Item, error) {
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(
			&it.ID, &it.URL, &it.Title, &it.Price, &it.Currency,
			&it.Priority, &it.Category, &it.Notes, &it.Status,
			&it.DesiredDate, &it.CreatedAt, &it.UpdatedAt,
			&it.PurchasedAt, &it.PriceConfirmed,
		); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, it)
	}
	return items, nil
}

// ─── Paydays ────────────────────────────────────────────────────────────────

func GetPaydays() ([]Payday, error) {
	rows, err := DB.Query(
		"SELECT id, frequency, day_of_month, day_of_week, interval_val, label, next_date, active, created_at FROM paydays ORDER BY active DESC, id ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("query paydays: %w", err)
	}
	defer rows.Close()

	var paydays []Payday
	for rows.Next() {
		var p Payday
		if err := rows.Scan(&p.ID, &p.Frequency, &p.DayOfMonth, &p.DayOfWeek, &p.Interval, &p.Label, &p.NextDate, &p.Active, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan payday: %w", err)
		}
		paydays = append(paydays, p)
	}
	return paydays, nil
}

func GetActivePaydays() ([]Payday, error) {
	rows, err := DB.Query(
		"SELECT id, frequency, day_of_month, day_of_week, interval_val, label, next_date, active, created_at FROM paydays WHERE active = 1 ORDER BY id ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("query active paydays: %w", err)
	}
	defer rows.Close()

	var paydays []Payday
	for rows.Next() {
		var p Payday
		if err := rows.Scan(&p.ID, &p.Frequency, &p.DayOfMonth, &p.DayOfWeek, &p.Interval, &p.Label, &p.NextDate, &p.Active, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan payday: %w", err)
		}
		paydays = append(paydays, p)
	}
	return paydays, nil
}

func CreatePayday(p *Payday) (int64, error) {
	res, err := DB.Exec(
		`INSERT INTO paydays (frequency, day_of_month, day_of_week, interval_val, label, next_date, active)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.Frequency, p.DayOfMonth, p.DayOfWeek, p.Interval, p.Label, p.NextDate, p.Active,
	)
	if err != nil {
		return 0, fmt.Errorf("create payday: %w", err)
	}
	return res.LastInsertId()
}

func UpdatePayday(p *Payday) error {
	_, err := DB.Exec(
		`UPDATE paydays SET frequency=?, day_of_month=?, day_of_week=?, interval_val=?, label=?, next_date=?, active=?
		 WHERE id=?`,
		p.Frequency, p.DayOfMonth, p.DayOfWeek, p.Interval, p.Label, p.NextDate, p.Active, p.ID,
	)
	return err
}

func DeletePayday(id int64) error {
	_, err := DB.Exec("DELETE FROM paydays WHERE id = ?", id)
	return err
}

// ─── Settings ───────────────────────────────────────────────────────────────

func GetSetting(key string) (string, error) {
	var val string
	err := DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func GetAllSettings() (map[string]string, error) {
	rows, err := DB.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		settings[k] = v
	}
	return settings, nil
}

func SetSetting(key, value string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

// ─── Budget Entries ─────────────────────────────────────────────────────────

func GetBudgetEntries() ([]BudgetEntry, error) {
	rows, err := DB.Query("SELECT id, amount, label, effective_date, created_at FROM budget_entries ORDER BY effective_date DESC")
	if err != nil {
		return nil, fmt.Errorf("query budget entries: %w", err)
	}
	defer rows.Close()

	var entries []BudgetEntry
	for rows.Next() {
		var e BudgetEntry
		if err := rows.Scan(&e.ID, &e.Amount, &e.Label, &e.EffectiveDate, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan budget entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func CreateBudgetEntry(e *BudgetEntry) (int64, error) {
	res, err := DB.Exec(
		"INSERT INTO budget_entries (amount, label, effective_date) VALUES (?, ?, ?)",
		e.Amount, e.Label, e.EffectiveDate,
	)
	if err != nil {
		return 0, fmt.Errorf("create budget entry: %w", err)
	}
	return res.LastInsertId()
}

func DeleteBudgetEntry(id int64) error {
	_, err := DB.Exec("DELETE FROM budget_entries WHERE id = ?", id)
	return err
}

func GetBudgetEntriesForDate(date string) ([]BudgetEntry, error) {
	rows, err := DB.Query(
		"SELECT id, amount, label, effective_date, created_at FROM budget_entries WHERE effective_date = ?",
		date,
	)
	if err != nil {
		return nil, fmt.Errorf("query budget entries for date: %w", err)
	}
	defer rows.Close()

	var entries []BudgetEntry
	for rows.Next() {
		var e BudgetEntry
		if err := rows.Scan(&e.ID, &e.Amount, &e.Label, &e.EffectiveDate, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan budget entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ─── Purchase Plan ──────────────────────────────────────────────────────────

func ClearPlan() error {
	_, err := DB.Exec("DELETE FROM purchase_plan")
	return err
}

func AddPlanEntry(p *PurchasePlan) (int64, error) {
	res, err := DB.Exec(
		`INSERT INTO purchase_plan (item_id, scheduled_date, payday_id, budget_cycle, rank, amount_allocated, status, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ItemID, p.ScheduledDate, p.PaydayID, p.BudgetCycle, p.Rank, p.AmountAllocated, p.Status, p.Notes,
	)
	if err != nil {
		return 0, fmt.Errorf("add plan entry: %w", err)
	}
	return res.LastInsertId()
}

func GetPlan() ([]PurchasePlan, error) {
	rows, err := DB.Query(
		`SELECT pp.id, pp.item_id, pp.scheduled_date, pp.payday_id, pp.budget_cycle,
		        pp.rank, pp.amount_allocated, pp.status, pp.created_at, pp.notes,
		        i.title, i.price, i.url
		 FROM purchase_plan pp
		 JOIN items i ON i.id = pp.item_id
		 ORDER BY pp.scheduled_date ASC, pp.rank ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query plan: %w", err)
	}
	defer rows.Close()

	var plan []PurchasePlan
	for rows.Next() {
		var p PurchasePlan
		if err := rows.Scan(
			&p.ID, &p.ItemID, &p.ScheduledDate, &p.PaydayID, &p.BudgetCycle,
			&p.Rank, &p.AmountAllocated, &p.Status, &p.CreatedAt, &p.Notes,
			&p.ItemTitle, &p.ItemPrice, &p.ItemURL,
		); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plan = append(plan, p)
	}
	return plan, nil
}

func MarkPlanItemCompleted(id int64) error {
	_, err := DB.Exec("UPDATE purchase_plan SET status='completed' WHERE id=?", id)
	return err
}

func MarkPlanItemSkipped(id int64) error {
	_, err := DB.Exec("UPDATE purchase_plan SET status='skipped' WHERE id=?", id)
	return err
}

// ─── Item Savings (for items being saved for across cycles) ─────────────────

func GetItemSavings(itemID int64) (float64, error) {
	var accumulated float64
	err := DB.QueryRow("SELECT COALESCE(accumulated, 0) FROM item_savings WHERE item_id = ?", itemID).Scan(&accumulated)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return accumulated, err
}

func UpsertItemSavings(itemID int64, accumulated float64) error {
	_, err := DB.Exec(
		`INSERT INTO item_savings (item_id, accumulated, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(item_id) DO UPDATE SET accumulated = ?, updated_at = datetime('now')`,
		itemID, accumulated, accumulated,
	)
	return err
}

func DeleteItemSavings(itemID int64) error {
	_, err := DB.Exec("DELETE FROM item_savings WHERE item_id = ?", itemID)
	return err
}

// ─── Notification Log ────────────────────────────────────────────────────────

func LogNotification(entry *NotificationLog) (int64, error) {
	res, err := DB.Exec(
		`INSERT INTO notification_log (item_id, type, channel, subject, body, delivered)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		entry.ItemID, entry.Type, entry.Channel, entry.Subject, entry.Body, entry.Delivered,
	)
	if err != nil {
		return 0, fmt.Errorf("log notification: %w", err)
	}
	return res.LastInsertId()
}

func GetNotificationsForItem(itemID int64, ntype string, date string) ([]NotificationLog, error) {
	rows, err := DB.Query(
		`SELECT id, item_id, type, channel, subject, body, sent_at, delivered
		 FROM notification_log
		 WHERE item_id = ? AND type = ? AND date(sent_at) = date(?)
		 ORDER BY sent_at DESC`,
		itemID, ntype, date,
	)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	var entries []NotificationLog
	for rows.Next() {
		var e NotificationLog
		if err := rows.Scan(&e.ID, &e.ItemID, &e.Type, &e.Channel, &e.Subject, &e.Body, &e.SentAt, &e.Delivered); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// GetPlanItemsReady returns plan entries where scheduled_date <= today and status = 'planned'.
// Used by the notification engine to find items ready to be purchased.
func GetPlanItemsReady() ([]PurchasePlan, error) {
	rows, err := DB.Query(
		`SELECT pp.id, pp.item_id, pp.scheduled_date, pp.payday_id, pp.budget_cycle,
		        pp.rank, pp.amount_allocated, pp.status, pp.created_at, pp.notes,
		        i.title, i.price, i.url
		 FROM purchase_plan pp
		 JOIN items i ON i.id = pp.item_id
		 WHERE pp.status = 'planned' AND pp.scheduled_date <= date('now')
		 ORDER BY pp.scheduled_date ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query ready plan items: %w", err)
	}
	defer rows.Close()

	var plan []PurchasePlan
	for rows.Next() {
		var p PurchasePlan
		if err := rows.Scan(
			&p.ID, &p.ItemID, &p.ScheduledDate, &p.PaydayID, &p.BudgetCycle,
			&p.Rank, &p.AmountAllocated, &p.Status, &p.CreatedAt, &p.Notes,
			&p.ItemTitle, &p.ItemPrice, &p.ItemURL,
		); err != nil {
			return nil, fmt.Errorf("scan ready plan: %w", err)
		}
		plan = append(plan, p)
	}
	return plan, nil
}
