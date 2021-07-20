package models

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/zsmartex/go-finex/config"
)

type TradingFee struct {
	ID         uint64 `gorm:"primaryKey"`
	MarketID   string
	Group      string
	Maker      decimal.Decimal
	Taker      decimal.Decimal
	MarketType string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Get trading fee for specific order that based on member group and market_id.
// TradingFee record selected with the next priorities:
//  1. both group and market_id match
//  2. group match
//  3. market_id match
//  4. both group and market_id are 'any'
//  5. default (zero fees)
func TradingFeeFor(group, market_type, market_id string) *TradingFee {
	var trading_fees []*TradingFee

	config.DataBase.Where(
		"\"market_id\" IN ? AND \"market_type\" IN ? AND \"group\" IN ?",
		[]string{market_id, "any"},
		[]string{market_type, "any"},
		[]string{group, "any"},
	).Find(&trading_fees)

	trading_fee := &TradingFee{}

	for _, tf := range trading_fees {
		if trading_fee.Weight() < tf.Weight() {
			trading_fee = tf
		}
	}

	return trading_fee
}

// Trading fee suitability expressed in weight.
// Trading fee with the greatest weight selected.
// Group match has greater weight then market_id match.
// E.g. Order for member with group 'vip-0' and market_id 'btcusd'
// (group == 'vip-0' && market_id == 'btcusd') >
// (group == 'vip-0' && market_id == 'any') >
// (group == 'any' && market_id == 'btcusd') >
// (group == 'any' && market_id == 'any')
func (t *TradingFee) Weight() int {
	var group_weight, market_weight int

	if t.Group == "any" {
		group_weight = 0
	} else {
		group_weight = 10
	}

	if t.MarketID == "any" {
		market_weight = 0
	} else {
		market_weight = 1
	}
	return group_weight + market_weight
}
