package engines

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/zsmartex/pkg"
	"github.com/zsmartex/pkg/order"
	"github.com/zsmartex/pkg/trade"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/shopspring/decimal"
	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/models"
	"github.com/zsmartex/finex/mq_client"
	"github.com/zsmartex/finex/types"
)

type TradeExecutorWorker struct {
	ExecutorMutex sync.RWMutex
}

type TradeExecutor struct {
	TradePayload *trade.Trade
	MakerOrder   *models.Order
	TakerOrder   *models.Order
}

func NewTradeExecutorWorker() *TradeExecutorWorker {
	return &TradeExecutorWorker{}
}

func (w *TradeExecutorWorker) Process(payload []byte) error {
	w.ExecutorMutex.Lock()
	defer w.ExecutorMutex.Unlock()

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
				"action": pkg.ActionSubmit,
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

func (t *TradeExecutor) IsMakerOrderFake() bool {
	return t.TradePayload.MakerOrder.IsFake()
}

func (t *TradeExecutor) IsTakerOrderFake() bool {
	return t.TradePayload.TakerOrder.IsFake()
}

func (t *TradeExecutor) VaildateTrade() error {
	var ask_order *models.Order
	var bid_order *models.Order
	var ask_order_fake bool
	var bid_order_fake bool

	if t.MakerOrder.Type == models.SideSell {
		ask_order_fake = t.TradePayload.MakerOrder.IsFake()
		bid_order_fake = t.TradePayload.TakerOrder.IsFake()
		ask_order = t.MakerOrder
		bid_order = t.TakerOrder
	} else {
		ask_order_fake = t.TradePayload.TakerOrder.IsFake()
		bid_order_fake = t.TradePayload.MakerOrder.IsFake()
		ask_order = t.TakerOrder
		bid_order = t.MakerOrder
	}

	if !ask_order_fake && ask_order.OrdType == types.TypeLimit && ask_order.Price.Decimal.GreaterThan(t.TradePayload.Price) {
		return fmt.Errorf("ask price exceeds strike price")
	} else if !bid_order_fake && bid_order.OrdType == types.TypeLimit && bid_order.Price.Decimal.LessThan(t.TradePayload.Price) {
		return fmt.Errorf("bid price is less than strike price")
	} else if !t.IsMakerOrderFake() && t.MakerOrder.State != models.StateWait {
		return fmt.Errorf("maker order state isn't equal to «wait» (%v)", t.MakerOrder.State)
	} else if !t.IsTakerOrderFake() && t.TakerOrder.State != models.StateWait {
		return fmt.Errorf("taker order state isn't equal to «wait» (%v)", t.TakerOrder.State)
	} else if !t.TradePayload.Total.IsPositive() {
		return fmt.Errorf("not enough funds")
	} else if !t.IsMakerOrderFake() && !t.IsTakerOrderFake() && decimal.Min(t.MakerOrder.Volume, t.TakerOrder.Volume).LessThan(t.TradePayload.Quantity) {
		return fmt.Errorf("not enough funds")
	} else if !t.IsMakerOrderFake() && t.MakerOrder.Volume.LessThan(t.TradePayload.Quantity) {
		return fmt.Errorf("not enough funds")
	} else if !t.IsTakerOrderFake() && t.TakerOrder.Volume.LessThan(t.TradePayload.Quantity) {
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

		if !t.IsMakerOrderFake() {
			tx.Clauses(clause.Locking{
				Strength: "UPDATE",
				Table:    clause.Table{Name: "orders"},
			}).Where("id = ?", t.TradePayload.MakerOrder.ID).First(t.MakerOrder)
		}
		if !t.IsTakerOrderFake() {
			tx.Clauses(clause.Locking{
				Strength: "UPDATE",
				Table:    clause.Table{Name: "orders"},
			}).Where("id = ?", t.TradePayload.TakerOrder.ID).First(t.TakerOrder)
		}

		if err := t.VaildateTrade(); err != nil {
			return err
		}

		// Check if accounts exists or create them.
		if !t.IsMakerOrderFake() {
			var af *models.Account // dont care
			config.DataBase.FirstOrCreate(&af, models.Account{
				MemberID:   t.MakerOrder.MemberID,
				CurrencyID: t.MakerOrder.IncomeCurrency().ID,
				Balance:    decimal.Zero,
				Locked:     decimal.Zero,
			})
		}

		if !t.IsTakerOrderFake() {
			var af *models.Account // dont care
			config.DataBase.FirstOrCreate(&af, models.Account{
				MemberID:   t.TakerOrder.MemberID,
				CurrencyID: t.TakerOrder.IncomeCurrency().ID,
				Balance:    decimal.Zero,
				Locked:     decimal.Zero,
			})
		}

		tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "accounts"},
		}).Where(
			"member_id IN ? AND currency_id IN ?",
			[]uint64{
				t.TradePayload.TakerOrder.MemberID,
				t.TradePayload.MakerOrder.MemberID,
			},
			[]string{
				market.BaseUnit,
				market.QuoteUnit,
			},
		).Find(&accounts)

		for _, account := range accounts {
			accounts_table[account.CurrencyID+":"+strconv.FormatUint(account.MemberID, 10)] = account
		}

		var side types.TakerType
		if t.TradePayload.TakerOrder.Side == order.SideSell {
			side = types.TypeSell
		} else {
			side = types.TypeBuy
		}

		trade = &models.Trade{
			Price:        t.TradePayload.Price,
			Amount:       t.TradePayload.Quantity,
			Total:        t.TradePayload.Total,
			MakerOrderID: t.TradePayload.MakerOrder.ID,
			TakerOrderID: t.TradePayload.TakerOrder.ID,
			MarketID:     t.TradePayload.Symbol,
			MakerID:      t.TradePayload.MakerOrder.MemberID,
			TakerID:      t.TradePayload.TakerOrder.MemberID,
			TakerType:    side,
		}

		if !t.IsMakerOrderFake() {
			if err := t.Strike(
				trade,
				t.MakerOrder,
				accounts_table[t.MakerOrder.OutcomeCurrency().ID+":"+strconv.FormatUint(t.MakerOrder.MemberID, 10)],
				accounts_table[t.MakerOrder.IncomeCurrency().ID+":"+strconv.FormatUint(t.MakerOrder.MemberID, 10)],
				tx,
			); err != nil {
				return err
			}
		}

		if !t.IsTakerOrderFake() {
			if err := t.Strike(
				trade,
				t.TakerOrder,
				accounts_table[t.TakerOrder.OutcomeCurrency().ID+":"+strconv.FormatUint(t.TakerOrder.MemberID, 10)],
				accounts_table[t.TakerOrder.IncomeCurrency().ID+":"+strconv.FormatUint(t.TakerOrder.MemberID, 10)],
				tx,
			); err != nil {
				return err
			}
		}

		if !t.IsMakerOrderFake() {
			tx.Save(t.MakerOrder)
		}

		if !t.IsTakerOrderFake() {
			tx.Save(t.TakerOrder)
		}
		config.DataBase.Create(trade)

		if !t.IsMakerOrderFake() && !t.IsTakerOrderFake() {
			trade.RecordCompleteOperations(t.TradePayload.SellOrder(), t.TradePayload.BuyOrder())
		}

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
	if !t.IsMakerOrderFake() {
		maker := trade.Maker()
		if payload_message, err := json.Marshal(trade.ToJSON(maker)); err == nil {
			mq_client.EnqueueEvent("private", maker.UID, "trade", payload_message)
		}
	}

	if !t.IsTakerOrderFake() {
		taker := trade.Taker()
		if payload_message, err := json.Marshal(trade.ToJSON(taker)); err == nil {
			mq_client.EnqueueEvent("private", taker.UID, "trade", payload_message)
		}
	}

	if payload_message, err := json.Marshal(map[string]interface{}{"trades": &[]models.TradeGlobalJSON{trade.TradeGlobalJSON()}}); err == nil {
		mq_client.EnqueueEvent("public", trade.MarketID, "trades", payload_message)
	}

	trade.WriteToInflux()
}
