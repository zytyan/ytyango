package q

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

type UnixTime struct {
	time.Time
}

func (u *UnixTime) Scan(value any) error {
	switch val := value.(type) {
	case int64:
		u.Time = time.Unix(val, 0)
		return nil
	case float64:
		u.Time = time.Unix(int64(val), 0)
		return nil
	default:
		return fmt.Errorf("cannot convert %v of type %T to UnixTime", value, value)
	}
}

func (u *UnixTime) Value() (driver.Value, error) {
	return u.Unix(), nil
}

type ChatCfg struct {
	ID             int64         `json:"id"`
	WebID          sql.NullInt64 `json:"web_id"`
	AutoCvtBili    bool          `json:"auto_cvt_bili"`
	AutoOcr        bool          `json:"auto_ocr"`
	AutoCalculate  bool          `json:"auto_calculate"`
	AutoExchange   bool          `json:"auto_exchange"`
	AutoCheckAdult bool          `json:"auto_check_adult"`
	SaveMessages   bool          `json:"save_messages"`
	EnableCoc      bool          `json:"enable_coc"`
	RespNsfwMsg    bool          `json:"resp_nsfw_msg"`
}
