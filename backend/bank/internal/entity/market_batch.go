package entity

import "time"

type MarketBatch struct {
	BatchID      int       `gorm:"primaryKey;autoIncrement;column:batch_id"`
	NewsSummary  string    `gorm:"type:text;column:news_summary"`
	InterestRate float32   `gorm:"column:interest_rate"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (MarketBatch) TableName() string {
	return "market_batches"
}
