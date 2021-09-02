package datatypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type IEOPaymentCurrency struct {
	Currency string          `json:"currency"`
	Bouns    decimal.Decimal `json:"bouns"`
}

// JSON defined JSON data type, need to implements driver.Valuer, sql.Scanner interface
type IEOPaymentCurrencies []IEOPaymentCurrency

// Value return json value, implement driver.Valuer interface
func (m IEOPaymentCurrencies) Value() (driver.Value, error) {
	data, err := json.Marshal(m)
	return string(data), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *IEOPaymentCurrencies) Scan(val interface{}) error {
	if val == nil {
		*m = IEOPaymentCurrencies{}
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := IEOPaymentCurrencies{}
	err := json.Unmarshal(ba, &t)
	*m = IEOPaymentCurrencies(t)
	return err
}

// GormDataType gorm common data type
func (m IEOPaymentCurrencies) GormDataType() string {
	return "jsonmap"
}

// GormDBDataType gorm db data type
func (IEOPaymentCurrencies) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}
