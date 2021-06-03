package workers

import (
	"encoding/json"
	"log"

	"github.com/zsmartex/go-finex/models"
	"github.com/zsmartex/go-finex/types"
)

type OrderProcessorWorker struct {
}

func NewOrderProcessorWorker() *OrderProcessorWorker {
	return &OrderProcessorWorker{}
}

func (w OrderProcessorWorker) Process(payload []byte) {
	var order_processor_payload *types.OrderProcessorPayloadMessage
	err := json.Unmarshal(payload, &order_processor_payload)
	if err != nil {
		log.Print(err)
	}

	order := order_processor_payload.Order

	switch order_processor_payload.Action {
	case types.ActionSubmit:
		models.SubmitOrder(order.ID)
	case types.ActionCancel:
		models.CancelOrder(order.ID)
	}
}
