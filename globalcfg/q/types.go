package q

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"
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

type ChatCfg struct {
	ID             int64         `json:"id"`
	WebID          sql.NullInt64 `json:"web_id"`
	AutoCvtBili    bool          `json:"auto_cvt_bili"  btnTxt:"自动转换Bilibili视频链接" pos:"1,1"`
	AutoOcr        bool          `json:"auto_ocr"`
	AutoCalculate  bool          `json:"auto_calculate" btnTxt:"自动计算算式" pos:"2,1"`
	AutoExchange   bool          `json:"auto_exchange"  btnTxt:"自动换算汇率" pos:"2,2"`
	AutoCheckAdult bool          `json:"auto_check_adult"`
	SaveMessages   bool          `json:"save_messages"  btnTxt:"保存群组消息" pos:"3,1"`
	EnableCoc      bool          `json:"enable_coc"     btnTxt:"启用CoC辅助" pos:"3,2"`
	RespNsfwMsg    bool          `json:"resp_nsfw_msg"  btnTxt:"响应来张色图" pos:"4,1"`
	Timezone       int64         `json:"timezone"`
}

func fromInnerCfg(cfg *chatCfg) *ChatCfg {
	return &ChatCfg{
		ID:             cfg.ID,
		WebID:          cfg.WebID,
		AutoCvtBili:    cfg.AutoCvtBili,
		AutoOcr:        cfg.AutoOcr,
		AutoCalculate:  cfg.AutoCalculate,
		AutoExchange:   cfg.AutoExchange,
		AutoCheckAdult: cfg.AutoCheckAdult,
		SaveMessages:   cfg.SaveMessages,
		EnableCoc:      cfg.EnableCoc,
		RespNsfwMsg:    cfg.RespNsfwMsg,
		Timezone:       cfg.Timezone,
	}
}
