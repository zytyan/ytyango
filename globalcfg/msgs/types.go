package msgs

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type UnixTime struct {
	time.Time
}

func (u *UnixTime) Scan(value any) error {
	switch val := value.(type) {
	case int64:
		// SQLite INTEGER
		u.Time = time.Unix(val, 0)
		return nil

	case float64:
		// JSON float → SQLC sometimes gives float
		u.Time = time.Unix(int64(val), 0)
		return nil

	case time.Time:
		// MySQL / Postgres
		u.Time = val
		return nil

	case []byte:
		// SQLite sometimes returns []byte for INTEGER
		i, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			return err
		}
		u.Time = time.Unix(i, 0)
		return nil

	case string:
		// If database stores as string timestamp
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		u.Time = time.Unix(i, 0)
		return nil
	}

	return fmt.Errorf("cannot convert %v (%T) to UnixTime", value, value)
}

func (u UnixTime) Value() (driver.Value, error) {
	return u.Unix(), nil
}

func (u UnixTime) ZapObject(name string) zap.Field {
	return zap.Time(name, u.Time)
}
