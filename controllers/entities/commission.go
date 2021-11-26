package entities

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type CommissionEntity struct {
	ID              int64             `json:"id"`
	AccountType     types.AccountType `json:"account_type"`
	MemberID        int64             `json:"member_id"`
	FriendUID       string            `json:"friend_uid"`
	EarnAmount      decimal.Decimal   `json:"earned_amount"`
	CurrencyID      string            `json:"currency_id"`
	ParentID        int64             `json:"parent_id"`
	ParentCreatedAt time.Time         `json:"parent_created_at"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}
