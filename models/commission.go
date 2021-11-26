package models

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type Commission struct {
	ID              int64
	AccountType     types.AccountType
	MemberID        int64
	FriendUID       string
	EarnAmount      decimal.Decimal
	CurrencyID      string
	ParentID        int64
	ParentCreatedAt time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
