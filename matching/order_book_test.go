package matching

import (
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/ericlagergren/decimal"
)

const instrument = "TEST"

func createOrder(id uint64, oType OrderType, params OrderParams, qty, price, stopPrice decimal.Big, side OrderSide) Order {
	return Order{
		ID:         id,
		Instrument: instrument,
		CustomerID: 1,
		Timestamp:  time.Now(),
		Type:       oType,
		Params:     params,
		Qty:        qty,
		FilledQty:  decimal.Big{},
		Price:      price,
		StopPrice:  stopPrice,
		Side:       side,
	}
}

func setup(coeff int64, exp int) (*TradeBook, *OrderBook) {
	tb := NewTradeBook(instrument)

	ob := NewOrderBook(instrument, decimal.New(coeff, exp), tb, NOPOrderRepository)
	return tb, ob
}

func TestOrderBook_MarketReject(t *testing.T) {
	_, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeMarket, 0, *decimal.New(5, 0), decimal.Big{}, decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeMarket, 0, *decimal.New(2, 0), decimal.Big{}, decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
}

func TestOrderBook_MarketToLimit(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeMarket, 0, *decimal.New(2, 0), decimal.Big{}, decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected match for market order, got no match")
	}
	if len(tb.trades) == 0 {
		t.Fatal("expected one trade, got none")
	}
	t.Logf("trade: %+v", tb.trades[0])
}

func TestOrderBook_LimitToMarket(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeMarket, 0, *decimal.New(2, 0), decimal.Big{}, decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected match for market order, got no match")
	}
	if len(tb.trades) == 0 {
		t.Fatal("expected one trade, got none")
	}
	t.Logf("trade: %+v", tb.trades[0])
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_No_Match(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2025, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Match(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Match_FullQty(t *testing.T) {
	tb, ob := setup(2025, -2)

	o1 := createOrder(1, TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideSell)
	matched, err := ob.Add(o1)
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	o2 := createOrder(2, TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy)
	matched, err = ob.Add(o2)
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
	if len(ob.activeOrders) != 0 {
		t.Errorf("expected 0 active orders, got %d", len(ob.activeOrders))
	}
}

func TestOrderBook_Limit_To_Limit_First_AON_Reject(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, *decimal.New(5, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_AON_Reject(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Both_AON_Reject(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, *decimal.New(2, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for  order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Both_AON(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, *decimal.New(5, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_First_AON(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamAON, *decimal.New(3, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	} else {
		t.Logf("trade: %+v", tb.trades[0])
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_AON(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, *decimal.New(3, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamAON, *decimal.New(2, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no match")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected a trade, got %d trades", len(tb.trades))
	} else {
		t.Logf("trade: %+v", tb.trades[0])
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_First_IOC_Reject(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, ParamIOC, *decimal.New(3, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if ob.orders.Asks.Len() != 0 {
		t.Fatalf("expected no asks, got %d", ob.orders.Asks.Len())
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	if len(tb.trades) != 0 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 1 {
		t.Errorf("expected 1 bid, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_IOC(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, *decimal.New(3, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamIOC, *decimal.New(2, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no matches")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 1 {
		t.Errorf("expected 1 ask, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
}

func TestOrderBook_Limit_To_Limit_Second_IOC_CancelCheck(t *testing.T) {
	tb, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, *decimal.New(3, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for this order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, ParamIOC, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got no matches")
	}
	if len(tb.trades) != 1 {
		t.Errorf("expected no trades, got %d trades", len(tb.trades))
	}
	if ob.orders.Asks.Len() != 0 {
		t.Errorf("expected 0 asks, got %d", ob.orders.Asks.Len())
	}
	if ob.orders.Bids.Len() != 0 {
		t.Errorf("expected 0 bids, got %d", ob.orders.Bids.Len())
	}
	order := ob.activeOrders[1]
	if !order.IsCancelled() {
		t.Log("IOC order should be cancelled after partial fill")
	}
	if order.FilledQty.Cmp(decimal.New(3, 0)) != 0 {
		t.Logf("expected filled qty for IOC order %d, got %v", 3, order.FilledQty)
	}
	t.Logf("%+v", order)
}

func TestOrderBook_Add_Bids(t *testing.T) {
	// test order sorting
	_, ob := setup(2025, -2)

	type orderData struct {
		Type      OrderType
		Params    OrderParams
		Qty       decimal.Big
		Price     decimal.Big
		StopPrice decimal.Big
		Side      OrderSide
	}

	data := [...]orderData{
		{TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2010, -2), decimal.Big{}, SideBuy},
		{TypeMarket, ParamAON, *decimal.New(11, 0), decimal.Big{}, decimal.Big{}, SideBuy},
		{TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2010, -2), decimal.Big{}, SideBuy},
		{TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2065, -2), decimal.Big{}, SideBuy},
		{TypeMarket, 0, *decimal.New(4, 0), decimal.Big{}, decimal.Big{}, SideBuy},
	}

	for i, d := range data {
		_, _ = ob.Add(createOrder(uint64(i+1), d.Type, d.Params, d.Qty, d.Price, d.StopPrice, d.Side))
	}

	sorted := []int{1, 4, 3, 0, 2}

	i := 0
	for iter := ob.orders.Bids.Iterator(); iter.Valid(); iter.Next() {
		order := ob.activeOrders[iter.Key().OrderID]

		expectedData := data[sorted[i]]

		priceEq := expectedData.StopPrice.Cmp(&order.Price)
		stopPriceEq := expectedData.StopPrice.Cmp(&order.StopPrice)

		equals := uint64(sorted[i]+1) == order.ID && expectedData.Type == order.Type && expectedData.Params == order.Params && expectedData.Qty.Cmp(&order.Qty) == 0 && priceEq == 0 && stopPriceEq == 0 && expectedData.Side == order.Side
		if !equals {
			t.Errorf("expected order ID %d to be in place %d, got a different order", sorted[i]+1, i)
		}

		i += 1
		t.Logf("%+v", order)
	}
}

func TestOrderBook_Add_Asks(t *testing.T) {
	// test order sorting
	_, ob := setup(2025, -2)

	type orderData struct {
		Type      OrderType
		Params    OrderParams
		Qty       decimal.Big
		Price     decimal.Big
		StopPrice decimal.Big
		Side      OrderSide
	}

	data := [...]orderData{
		{TypeLimit, 0, *decimal.New(7, 0), *decimal.New(2000, -2), decimal.Big{}, SideSell},
		{TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2013, -2), decimal.Big{}, SideSell},
		{TypeLimit, 0, *decimal.New(8, 0), *decimal.New(2000, -2), decimal.Big{}, SideSell},
		{TypeMarket, 0, *decimal.New(9, 0), decimal.Big{}, decimal.Big{}, SideSell},
		{TypeLimit, 0, *decimal.New(3, 0), *decimal.New(2055, -2), decimal.Big{}, SideSell},
	}

	for i, d := range data {
		_, _ = ob.Add(createOrder(uint64(i+1), d.Type, d.Params, d.Qty, d.Price, d.StopPrice, d.Side))
	}

	sorted := []int{3, 0, 2, 1, 4}

	i := 0
	for iter := ob.orders.Asks.Iterator(); iter.Valid(); iter.Next() {
		order := ob.activeOrders[iter.Key().OrderID]

		expectedData := data[sorted[i]]

		priceEq := expectedData.StopPrice.Cmp(&order.Price)
		stopPriceEq := expectedData.StopPrice.Cmp(&order.StopPrice)

		equals := uint64(sorted[i]+1) == order.ID && expectedData.Type == order.Type && expectedData.Params == order.Params && expectedData.Qty.Cmp(&order.Qty) == 0 && priceEq == 0 && stopPriceEq == 0 && expectedData.Side == order.Side
		if !equals {
			t.Errorf("expected order ID %d to be in place %d, got a different order", sorted[i]+1, i)
		}

		i += 1
		t.Logf("%+v", order)
	}
}

func TestOrderBook_Add_MarketPrice_Change(t *testing.T) {
	_, ob := setup(2025, -2)

	matched, err := ob.Add(createOrder(1, TypeLimit, 0, *decimal.New(2, 0), *decimal.New(2010, -2), decimal.Big{}, SideSell))
	if err != nil {
		t.Error(err)
	}
	if matched {
		t.Errorf("expected no match for market order, got a match")
	}
	matched, err = ob.Add(createOrder(2, TypeLimit, 0, *decimal.New(5, 0), *decimal.New(2012, -2), decimal.Big{}, SideBuy))
	if err != nil {
		t.Error(err)
	}
	if !matched {
		t.Errorf("expected a match for this order, got a match")
	}
	if ob.marketPrice.Cmp(decimal.New(2012, -2)) != 0 {
		t.Errorf("expected market price to be %f, got %s", 20.12, ob.marketPrice.String())
	}
}

func BenchmarkOrderBook_Add(b *testing.B) {
	ballast := make([]byte, 1<<32) // 1GB of memory ballast, to reduce round trips to the kernel
	_ = ballast

	var match bool
	var err error
	tb, ob := setup(2025, -2)

	orders := make([]Order, b.N)
	for i := range orders {
		order := createRandomOrder(i + 1)
		orders[i] = order
	}
	b.Logf("b.N: %d bids: %d asks: %d orders: %d ", b.N, ob.orders.Bids.Len(), ob.orders.Asks.Len(), len(ob.activeOrders))

	measureMemory(b)
	b.ReportAllocs()

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		match, err = ob.Add(orders[i])
	}
	b.StopTimer()

	_ = match
	_ = err
	b.Logf("orders len: %d bids len: %d asks len: %d trades len: %d", len(ob.activeOrders), ob.orders.Bids.Len(), ob.orders.Asks.Len(), len(tb.trades))

	expectedOrderLen := ob.orders.Len(SideBuy) + ob.orders.Len(SideSell)
	if len(ob.activeOrders) != expectedOrderLen {
		b.Errorf("expected %d active orders, got %d", expectedOrderLen, len(ob.activeOrders))
	}

	measureMemory(b)
}

func measureMemory(b *testing.B) {
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	b.Logf("total: %dB stack: %dB GCCPUFraction: %f total heap alloc: %dB", endMem.TotalAlloc,
		endMem.StackInuse, endMem.GCCPUFraction, endMem.HeapAlloc)
	b.Logf("alloc: %dB heap inuse: %dB", endMem.Alloc, endMem.HeapInuse)
}

func createRandomOrder(i int) Order {
	isMarket := rand.Int()%20 == 0
	isBuy := rand.Int()%2 == 0
	isAON := rand.Int()%20 == 0
	isIOC := rand.Int()%25 == 0

	qty := decimal.New(int64(2025+rand.Intn(200)-100), 3)
	fPrice := 2025 + rand.Intn(200) - 100
	if isBuy {
		fPrice += rand.Intn(150)
	}
	price := decimal.New(int64(fPrice), -2)

	oType := TypeLimit
	if isMarket {
		price = decimal.New(0, 0)
		oType = TypeMarket
	}
	var params OrderParams
	if isAON {
		//params |= ParamAON
	}
	if isIOC {
		//params |= ParamIOC
	}
	oSide := SideSell
	if isBuy {
		oSide = SideBuy
	}

	order := Order{
		ID:         uint64(i + 1),
		Instrument: instrument,
		CustomerID: 1,
		Timestamp:  time.Now(),
		Type:       oType,
		Params:     params,
		Qty:        *qty,
		FilledQty:  decimal.Big{},
		Price:      *price,
		StopPrice:  *price.Add(price, decimal.New(10, 0)),
		Side:       oSide,
		Cancelled:  false,
	}
	return order
}
