package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type ReleaseCommission struct {
	ID          uint64
	AccountType types.AccountType
	MemberID    uint64
	EarnedBTC   decimal.Decimal
	FriendTrade uint64
	Friend      uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
