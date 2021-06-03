package workers

import (
	"encoding/json"
	"errors"
	"log"
	"math"

	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/matching"
	"github.com/zsmartex/go-finex/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TradeExecutorWorker struct {
}

type TradeExecutor struct {
	TradePayload *matching.Trade
	MakerOrder   models.Order
	TakerOrder   models.Order
}

func NewTradeExecutorWorker() *TradeExecutorWorker {
	return &TradeExecutorWorker{}
}

func (w *TradeExecutorWorker) Process(payload []byte) {
	trade_executor := &TradeExecutor{}

	if err := json.Unmarshal(payload, &trade_executor.TradePayload); err != nil {
		log.Print(err)
		return
	}

	trade, err := trade_executor.CreateTradeAndStrikeOrders()

	if err != nil {
		log.Print(err)
		return
	}

	trade_executor.PublishTrade(trade)
}

func (t *TradeExecutor) CreateTradeAndStrikeOrders() (models.Trade, error) {
	var trade models.Trade

	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		var market models.Market
		var makerOutcomeAccount models.Account
		var makerIncomeAccount models.Account
		var takerOutcomeAccount models.Account
		var takerIncomeAccount models.Account
		price, _ := t.TradePayload.Price.Float64()
		amount, _ := t.TradePayload.Qty.Float64()
		total, _ := t.TradePayload.Total.Float64()

		config.DataBase.Find(&market, t.TradePayload.Instrument)

		tx.Clauses(clause.Locking{Strength: "UPDATE"})
		tx.Where("id = ?", t.TradePayload.MakerOrderID).Find(&t.MakerOrder)
		tx.Where("id = ?", t.TradePayload.TakerOrderID).Find(&t.TakerOrder)

		t.VaildateTrade()
		tx.Where("member_id = ? AND currency_id = ?", t.MakerOrder.MemberID, market.BaseUnit).Find(&makerOutcomeAccount)
		tx.Where("member_id = ? AND currency_id = ?", t.MakerOrder.MemberID, market.QuoteUnit).Find(&makerIncomeAccount)
		tx.Where("member_id = ? AND currency_id = ?", t.TakerOrder.MemberID, market.BaseUnit).Find(&takerOutcomeAccount)
		tx.Where("member_id = ? AND currency_id = ?", t.TakerOrder.MemberID, market.QuoteUnit).Find(&takerIncomeAccount)

		trade := models.Trade{
			Price:        price,
			Amount:       amount,
			Total:        total,
			MakerOrderID: t.TradePayload.MakerOrderID,
			TakerOrderID: t.TradePayload.TakerOrderID,
			MakerID:      t.TradePayload.MakerID,
			TakerID:      t.TradePayload.TakerID,
			TakerType:    t.TakerOrder.Side(),
		}
		tx.Create(&trade)

		t.Strike(&trade, &t.TakerOrder, &makerOutcomeAccount, &makerIncomeAccount, tx)
		t.Strike(&trade, &t.TakerOrder, &takerOutcomeAccount, &takerIncomeAccount, tx)
		trade.RecordCompleteOperations()

		// return nil will commit the whole transaction
		return nil
	})

	return trade, err
}

func (t *TradeExecutor) VaildateTrade() error {
	var ask_order models.Order
	var bid_order models.Order
	price, _ := t.TradePayload.Price.Float64()
	amount, _ := t.TradePayload.Qty.Float64()
	total, _ := t.TradePayload.Total.Float64()

	if t.MakerOrder.Type == models.SideSell {
		ask_order = t.MakerOrder
		bid_order = t.TakerOrder
	} else {
		ask_order = t.TakerOrder
		bid_order = t.MakerOrder
	}

	if ask_order.OrdType == matching.TypeLimit && *ask_order.Price > price {
		return errors.New("ask price exceeds strike price")
	} else if bid_order.OrdType == matching.TypeLimit && *bid_order.Price < price {
		return errors.New("bid price is less than strike price")
	} else if t.MakerOrder.State != models.StateWait {
		return errors.New("Maker order state isn't equal to «wait» (" + string(t.MakerOrder.State) + ").")
	} else if t.TakerOrder.State != models.StateWait {
		return errors.New("Taker order state isn't equal to «wait» (" + string(t.TakerOrder.State) + ").")
	} else if total <= 0 || math.Min(t.MakerOrder.Volume, t.TakerOrder.Volume) < amount {
		return errors.New("not enough funds")
	}

	return nil
}

func (t *TradeExecutor) Strike(trade *models.Trade, order *models.Order, outcome_account, income_account *models.Account, tx *gorm.DB) {
	var outcome_value, income_value float64
	if order.Type == models.SideSell {
		outcome_value = trade.Amount
		income_value = trade.Total
	} else {
		outcome_value = trade.Total
		income_value = trade.Amount
	}
	fee := income_value * trade.OrderFee(*order)
	real_income_value := income_value - fee

	outcome_account.UnlockAndSubFunds(tx, outcome_value)
	income_account.PlusFunds(tx, real_income_value)

	order.Volume -= trade.Amount
	order.Locked -= outcome_value
	order.FundsReceived += income_value
	order.TradesCount += 1

	if order.Volume == 0 {
		order.State = models.StateDone

		// Unlock not used funds.
		if order.Locked != 0 {
			outcome_account.UnlockFunds(tx, order.Locked)
		}
	} else if order.OrdType == matching.TypeMarket && order.Locked == 0 {
		order.State = models.StateCancel
		order.RecordCancelOperations()
	}
}

func (t *TradeExecutor) PublishTrade(trade models.Trade) {
	trade.TriggerEvent()
	trade.WriteToInflux()
	t.MakerOrder.TriggerEvent()
	t.TakerOrder.TriggerEvent()
}
