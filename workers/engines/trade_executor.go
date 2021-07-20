package engines

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/matching"
	"github.com/zsmartex/go-finex/models"
	"github.com/zsmartex/go-finex/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TradeExecutorWorker struct {
}

type TradeExecutor struct {
	TradePayload *matching.Trade
	MakerOrder   *models.Order
	TakerOrder   *models.Order
}

func NewTradeExecutorWorker() *TradeExecutorWorker {
	return &TradeExecutorWorker{}
}

func (w *TradeExecutorWorker) Process(payload []byte) error {
	trade_executor := &TradeExecutor{
		MakerOrder: &models.Order{},
		TakerOrder: &models.Order{},
	}

	if err := json.Unmarshal(payload, &trade_executor.TradePayload); err != nil {
		return err
	}

	trade, err := trade_executor.CreateTradeAndStrikeOrders()

	if err != nil {
		for _, order := range []*models.Order{trade_executor.MakerOrder, trade_executor.TakerOrder} {
			if order.State != models.StateWait {
				continue
			}

			matching_payload_message, err := json.Marshal(map[string]interface{}{
				"action": matching.ActionSubmit,
				"order":  order.ToMatchingAttributes(),
			})

			if err != nil {
				continue
			}

			config.Nats.Publish("matching", matching_payload_message)
		}
		return err
	}

	trade_executor.PublishTrade(trade)
	return nil
}

func (t *TradeExecutor) VaildateTrade() error {
	var ask_order *models.Order
	var bid_order *models.Order

	if t.MakerOrder.Type == models.SideSell {
		ask_order = t.MakerOrder
		bid_order = t.TakerOrder
	} else {
		ask_order = t.TakerOrder
		bid_order = t.MakerOrder
	}

	if ask_order.OrdType == types.TypeLimit && ask_order.Price.Decimal.GreaterThan(t.TradePayload.Price) {
		return fmt.Errorf("ask price exceeds strike price")
	} else if bid_order.OrdType == types.TypeLimit && bid_order.Price.Decimal.LessThan(t.TradePayload.Price) {
		return fmt.Errorf("bid price is less than strike price")
	} else if t.MakerOrder.State != models.StateWait {
		return fmt.Errorf("maker order state isn't equal to «wait» (%v)", t.MakerOrder.State)
	} else if t.TakerOrder.State != models.StateWait {
		return fmt.Errorf("taker order state isn't equal to «wait» (%v)", t.TakerOrder.State)
	} else if !t.TradePayload.Total.IsPositive() || decimal.Min(t.MakerOrder.Volume, t.TakerOrder.Volume).LessThan(t.TradePayload.Quantity) {
		return fmt.Errorf("not enough funds")
	}

	return nil
}

func (t *TradeExecutor) CreateTradeAndStrikeOrders() (*models.Trade, error) {
	var trade *models.Trade

	err := config.DataBase.Transaction(func(tx *gorm.DB) error {
		var accounts []*models.Account
		var market *models.Market
		accounts_table := make(map[string]*models.Account)

		config.DataBase.First(&market, "symbol = ?", t.TradePayload.Symbol)

		tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "orders"},
		}).Where("id = ?", t.TradePayload.MakerOrderID).First(t.MakerOrder)
		tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "orders"},
		}).Where("id = ?", t.TradePayload.TakerOrderID).First(t.TakerOrder)

		if err := t.VaildateTrade(); err != nil {
			return err
		}

		tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "accounts"},
		}).Where("member_id IN ? AND currency_id IN ?", []uint64{t.MakerOrder.MemberID, t.MakerOrder.MemberID}, []string{market.BaseUnit, market.QuoteUnit}).Find(&accounts)

		for _, account := range accounts {
			accounts_table[account.CurrencyID+":"+strconv.FormatUint(account.MemberID, 10)] = account
		}

		trade = &models.Trade{
			Price:        t.TradePayload.Price,
			Amount:       t.TradePayload.Quantity,
			Total:        t.TradePayload.Total,
			MakerOrderID: t.TradePayload.MakerOrderID,
			TakerOrderID: t.TradePayload.TakerOrderID,
			MarketID:     t.TradePayload.Symbol,
			MakerID:      t.TradePayload.MakerID,
			TakerID:      t.TradePayload.TakerID,
			TakerType:    t.TakerOrder.Side(),
		}
		config.DataBase.Create(trade)

		if err := t.Strike(
			trade,
			t.MakerOrder,
			accounts_table[t.MakerOrder.OutcomeCurrency().ID+":"+strconv.FormatUint(t.MakerOrder.MemberID, 10)],
			accounts_table[t.MakerOrder.IncomeCurrency().ID+":"+strconv.FormatUint(t.MakerOrder.MemberID, 10)],
			tx,
		); err != nil {
			return err
		}
		if err := t.Strike(
			trade,
			t.TakerOrder,
			accounts_table[t.TakerOrder.OutcomeCurrency().ID+":"+strconv.FormatUint(t.TakerOrder.MemberID, 10)],
			accounts_table[t.TakerOrder.IncomeCurrency().ID+":"+strconv.FormatUint(t.TakerOrder.MemberID, 10)],
			tx,
		); err != nil {
			return err
		}

		tx.Save(t.MakerOrder)
		tx.Save(t.TakerOrder)

		trade.RecordCompleteOperations()

		// return nil will commit the whole transaction
		return nil
	})

	return trade, err
}

func (t *TradeExecutor) Strike(trade *models.Trade, order *models.Order, outcome_account, income_account *models.Account, tx *gorm.DB) error {
	var outcome_value, income_value decimal.Decimal
	if order.Type == models.SideSell {
		outcome_value = trade.Amount
		income_value = trade.Total
	} else {
		outcome_value = trade.Total
		income_value = trade.Amount
	}
	fee := income_value.Mul(trade.OrderFee(order))
	real_income_value := income_value.Sub(fee)

	if err := outcome_account.UnlockAndSubFunds(tx, outcome_value); err != nil {
		return err
	}
	if err := income_account.PlusFunds(tx, real_income_value); err != nil {
		return err
	}

	order.Volume = order.Volume.Sub(trade.Amount)
	order.Locked = order.Locked.Sub(outcome_value)
	order.FundsReceived = income_value.Add(order.FundsReceived)
	order.TradesCount += 1

	if order.Volume.IsZero() {
		order.State = models.StateDone

		// Unlock not used funds.
		if !order.Locked.IsZero() {
			if err := outcome_account.UnlockFunds(tx, order.Locked); err != nil {
				return err
			}
		}
	} else if order.OrdType == types.TypeMarket && order.Locked.IsZero() {
		order.State = models.StateCancel
		order.RecordCancelOperations()
	}

	return nil
}

func (t *TradeExecutor) PublishTrade(trade *models.Trade) {
	trade.TriggerEvent()
	trade.WriteToInflux()
}
