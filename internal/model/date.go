package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const DateLayout = "2006-01-02"

// Date is a calendar date without time or timezone, serialized as "YYYY-MM-DD".
type Date struct {
	time.Time
}

func ParseDate(s string) (Date, error) {
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		return Date{}, err
	}

	return Date{t}, nil
}

func (d Date) String() string { return d.Format(DateLayout) }

func (d Date) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// Scan implements sql.Scanner so pgx can read a DATE column.
func (d *Date) Scan(src any) error {
	t, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("model.Date: cannot scan %T", src)
	}

	d.Time = t

	return nil
}

// Value implements driver.Valuer so pgx can write a DATE column.
func (d Date) Value() (driver.Value, error) { return d.Format(DateLayout), nil }
