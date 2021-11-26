package models

import "time"

type OperationType = string
type OperationScope = string

var (
	TypeLiability OperationType = "liability"
	TypeAsset     OperationType = "asset"
	TypeExpense   OperationType = "expense"
	TypeRevenue   OperationType = "revenue"
)

var (
	ScopeMember   OperationScope = "member"
	ScopePlatform OperationScope = "platform"
)

type OperationsAccount struct {
	ID           int64
	Code         int32
	Type         OperationType
	Kind         string
	CurrencyType CurrencyType
	Description  string
	Scope        OperationScope
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
