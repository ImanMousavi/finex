package entities

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type ReleaseCommissionEntity struct {
	ID          int64             `json:"id"`
	AccountType types.AccountType `json:"account_type"`
	MemberID    int64             `json:"member_id"`
	EarnedBTC   decimal.Decimal   `json:"earned_btc"`
	FriendTrade int64             `json:"friend_trade"`
	Friend      int64             `json:"friend"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
