package tracker

import (
	"log"

	"github.com/CaptDany/kestrel/internal/db"
)

func recordPrice(itemID int64, price *float64, currency string) {
	if price == nil {
		return
	}
	if _, err := db.RecordPriceHistory(&db.PriceHistory{
		ItemID:   itemID,
		Price:    price,
		Currency: currency,
	}); err != nil {
		log.Printf("tracker: record price for item %d: %v", itemID, err)
	}
}

func lowestPrice(itemID int64) *float64 {
	p, err := db.GetLowestPrice(itemID)
	if err != nil {
		log.Printf("tracker: get lowest price for item %d: %v", itemID, err)
		return nil
	}
	return p
}


