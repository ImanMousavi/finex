package cron

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/types"
)

type GlobalPriceJob struct {
}

func (j *GlobalPriceJob) Process() {
	var global_price types.GlobalPrice

	resp, err := http.Get("https://min-api.cryptocompare.com/data/pricemulti?fsyms=USD,USDT&tsyms=USD,USDT,EUR,VND,CNY,JPY")
	if err != nil {
		config.Logger.Error(err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		config.Logger.Error(err.Error())
		return
	}
	// Convert the body to type string
	if err := json.Unmarshal(body, &global_price); err != nil {
		config.Logger.Error(err.Error())
		return
	}

	if err := config.Redis.Set("finex:h24:global_price", string(body), 10*time.Minute); err != nil {
		config.Logger.Error(err.Error())
		return
	}

	time.Sleep(10 * time.Minute)
}
