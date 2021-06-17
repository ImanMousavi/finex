package engines

import (
	"encoding/json"

	"github.com/zsmartex/go-finex/config"
	"github.com/zsmartex/go-finex/matching"
	"github.com/zsmartex/go-finex/models"
)

type OrderProcessorPayloadMessage struct {
	Action matching.PayloadAction `json:"action"`
	Order  matching.Order         `json:"order"`
}

type OrderProcessorWorker struct {
}

func NewOrderProcessorWorker() *OrderProcessorWorker {
	kclass := &OrderProcessorWorker{}

	var orders []models.Order
	config.DataBase.Where("state = ?", models.StatePending).Find(&orders)
	for _, order := range orders {
		if err := models.SubmitOrder(order.ID); err != nil {
			config.Logger.Errorf("Error: %s", err.Error())
			break
		}
	}

	return kclass
}

func (w OrderProcessorWorker) Process(payload []byte) error {
	var order_processor_payload OrderProcessorPayloadMessage
	err := json.Unmarshal(payload, &order_processor_payload)

	if err != nil {
		return err
	}

	order := order_processor_payload.Order

	switch order_processor_payload.Action {
	case matching.ActionSubmit:
		err = models.SubmitOrder(order.ID)
	case matching.ActionCancel:
		err = models.CancelOrder(order.ID)
	}

	if err != nil {
		return err
	}

	return nil
}
