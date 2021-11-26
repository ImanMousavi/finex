package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type ReleaseCommission struct {
	ID          int64
	AccountType types.AccountType
	MemberID    int64
	EarnedBTC   decimal.Decimal
	FriendTrade int64
	Friend      int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
