package entities

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/types"
)

type ReleaseCommissionEntity struct {
	ID          uint64            `json:"id"`
	AccountType types.AccountType `json:"account_type"`
	MemberID    uint64            `json:"member_id"`
	EarnedBTC   decimal.Decimal   `json:"earned_btc"`
	FriendTrade uint64            `json:"friend_trade"`
	Friend      uint64            `json:"friend"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
