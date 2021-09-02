package engines

import (
	"encoding/json"

	"github.com/zsmartex/finex/config"
	"github.com/zsmartex/finex/models"
)

type IEOOrderProcessorWorker struct {
}

func NewIEOOrderProcessorWorker() *IEOOrderProcessorWorker {
	kclass := &IEOOrderProcessorWorker{}

	var ieo_orders []*models.IEOOrder
	config.DataBase.Find(&ieo_orders, "state = ?", models.StatePending)
	for _, ieo_order := range ieo_orders {
		if err := models.SubmitIEOOrder(ieo_order.ID); err != nil {
			config.Logger.Errorf("Error: %s", err.Error())
			break
		}
	}

	return kclass
}

func (w *IEOOrderProcessorWorker) Process(payload []byte) error {
	var payload_ieo_order_message *models.IEOOrder
	if err := json.Unmarshal(payload, &payload_ieo_order_message); err != nil {
		return err
	}

	if err := models.SubmitIEOOrder(payload_ieo_order_message.ID); err != nil {
		return err
	}

	return nil
}
