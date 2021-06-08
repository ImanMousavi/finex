package cron

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/types"
)

type GlobalPriceJob struct {
}

func (j *GlobalPriceJob) Process() {
	var global_price types.GlobalPrice

	resp, err := http.Get("https://min-api.cryptocompare.com/data/pricemulti?fsyms=USD,USDT&tsyms=USD,USDT,EUR,VND,CNY,JPY")
	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return
	}
	// Convert the body to type string
	if err := json.Unmarshal(body, &global_price); err != nil {
		log.Fatalln(err)
		return
	}

	config.Redis.SetKey("finex:h24:global_price", global_price, redis.KeepTTL)

	time.Sleep(10 * time.Minute)
}
