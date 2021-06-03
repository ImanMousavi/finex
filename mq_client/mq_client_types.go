package mq_client

type Exchange struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type Queue struct {
	Name    string `yaml:"name"`
	Durable bool   `yaml:"durable"`
}

type Binding struct {
	Queue      string `yaml:"queue"`
	CleanStart bool   `yaml:"clean_start"`
	Exchange   string `yaml:"exchange"`
}

type Channel struct {
	Prefetch int `yaml:"prefetch"`
}

type MQClientConfig struct {
	Exchange struct {
		Trade        Exchange `yaml:"trade"`
		Notification Exchange `yaml:"notification"`
		Orderbook    Exchange `yaml:"orderbook"`
		Events       Exchange `yaml:"events"`
		Matching     Exchange `yaml:"matching"`
	}
	Queue struct {
		Matching              Queue `yaml:"matching"`
		NewTrade              Queue `yaml:"new_trade"`
		OrderProcessor        Queue `yaml:"order_processor"`
		DepthCache            Queue `yaml:"depth_cache"`
		MarketTicker          Queue `yaml:"market_ticker"`
		WithdrawCoin          Queue `yaml:"withdraw_coin"`
		DepositCollectionFees Queue `yaml:"deposit_collection_fees"`
		DepositCollection     Queue `yaml:"deposit_collection"`
		DepositCoinAddress    Queue `yaml:"deposit_coin_address"`
		InfluxWriter          Queue `yaml:"influx_writer"`
		TradeError            Queue `yaml:"trade_error"`
		EventsProcessor       Queue `yaml:"events_processor"`
	}
	Binding struct {
		Matching           Binding `yaml:"matching"`
		TradeExecutor      Binding `yaml:"trade_executor"`
		OrderProcessor     Binding `yaml:"order_processor"`
		WithdrawCoin       Binding `yaml:"withdraw_coin"`
		DepositCoinAddress Binding `yaml:"deposit_coin_address"`
		InfluxWriter       Binding `yaml:"influx_writer"`
		TradeError         Binding `yaml:"trade_error"`
		EventsProcessor    Binding `yaml:"events_processor"`
		DepthCache         Binding `yaml:"depth_cache"`
	}
	Channel struct {
		TradeExecutor  Channel `yaml:"trade_executor"`
		OrderProcessor Channel `yaml:"order_processor"`
		Matching       Channel `yaml:"matching"`
		DepthCache     Channel `yaml:"depth_cache"`
	}
}
