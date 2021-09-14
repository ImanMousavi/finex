package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Commission struct {
	ID              uint64
	AccountType     string
	MemberID        uint64
	FriendUID       string
	EarnAmount      decimal.Decimal
	CurrencyID      string
	ParentID        uint64
	ParentCreatedAt time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
