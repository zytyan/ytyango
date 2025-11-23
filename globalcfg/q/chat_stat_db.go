package q

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
)

type UserMsgStat struct {
	MsgCount int64
	MsgLen   int64
}

type UserMsgStatMap map[int64]*UserMsgStat

func (u *UserMsgStatMap) Scan(src any) error {
	data, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("src must be []byte, but got %T", src)
	}
	if len(data) == 0 {
		*u = make(UserMsgStatMap)
		return nil
	}
	return gob.NewDecoder(bytes.NewReader(data)).Decode(u)

}
func (u UserMsgStatMap) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(u)
	return buf.Bytes(), err
}

type TenMinuteStats [24 * 6]int64

func (t *TenMinuteStats) Scan(src any) error {
	data, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("src must be []byte, but got %T", src)
	}
	if len(data) == 0 {
		*t = TenMinuteStats{}
		return nil
	}
	return gob.NewDecoder(bytes.NewReader(data)).Decode(t)
}

func (t TenMinuteStats) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(t)
	return buf.Bytes(), err
}

type ChatStat struct {
	mu       sync.Mutex
	timezone int64
	ChatStatDaily
}

func (s *ChatStat) IncMessage(userId, txtLen, unixTime, messageId int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.MessageCount++
	if s.UserMsgStat == nil {
		s.UserMsgStat = make(UserMsgStatMap, 4)
	}
	stat, ok := s.UserMsgStat[userId]
	if !ok || stat == nil {
		stat = &UserMsgStat{}
		s.UserMsgStat[userId] = stat
	}
	const daySeconds = 24 * 60 * 60
	const tenMinutes = 10 * 60
	timeSec := (unixTime + s.timezone) % daySeconds
	if timeSec < 0 {
		timeSec += daySeconds
	}
	idx := int(timeSec / tenMinutes)
	s.MsgCountByTime[idx]++
	if s.MsgIDAtTimeStart[idx] == 0 {
		s.MsgIDAtTimeStart[idx] = messageId
	}
	stat.MsgCount++
	stat.MsgLen += txtLen
	s.mu.Unlock()
}

func (s *ChatStat) IncPhotoCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.PhotoCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncVideoCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.VideoCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncStickerCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.StickerCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncForwardCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.ForwardCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncMarsCount(maxMarsCount int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.MarsCount++
	s.MaxMarsCount = max(maxMarsCount, s.MaxMarsCount)
	s.mu.Unlock()
}
func (s *ChatStat) IncRacyCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.RacyCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncAdultCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.AdultCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncDownloadVideoCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.DownloadVideoCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncDownloadAudioCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.DownloadAudioCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncDioAddUserCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.DioAddUserCount++
	s.mu.Unlock()
}
func (s *ChatStat) IncDioBanUserCount() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.DioBanUserCount++
	s.mu.Unlock()
}

func (s *ChatStat) Save(ctx context.Context, q *Queries) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return q.UpdateChatStatDaily(ctx, UpdateChatStatDailyParams{
		MessageCount:       s.MessageCount,
		PhotoCount:         s.PhotoCount,
		VideoCount:         s.VideoCount,
		StickerCount:       s.StickerCount,
		ForwardCount:       s.ForwardCount,
		MarsCount:          s.MarsCount,
		MaxMarsCount:       s.MaxMarsCount,
		RacyCount:          s.RacyCount,
		AdultCount:         s.AdultCount,
		DownloadVideoCount: s.DownloadVideoCount,
		DownloadAudioCount: s.DownloadAudioCount,
		DioAddUserCount:    s.DioAddUserCount,
		DioBanUserCount:    s.DioBanUserCount,
		UserMsgStat:        s.UserMsgStat,
		MsgCountByTime:     s.MsgCountByTime,
		MsgIDAtTimeStart:   s.MsgIDAtTimeStart,
		ChatID:             s.ChatID,
		StatDate:           s.StatDate,
	})
}

var m = xsync.NewMapOf[int64, *ChatStat]()

func (q *Queries) getOrCreateChatStat(ctx context.Context, chatId int64, day int64) (ChatStatDaily, error) {
	daily, err := q.getChatStat(ctx, chatId, day)
	if errors.Is(err, sql.ErrNoRows) {
		return q.createChatStatDaily(ctx, chatId, day)
	}
	if err != nil {
		return ChatStatDaily{}, err
	}
	return daily, err

}

func (q *Queries) chatStatAtWithTimezone(ctx context.Context, chatId, unixTime, timezone int64) (*ChatStat, error) {
	const daySeconds = 24 * 60 * 60
	day := (unixTime + timezone) / daySeconds
	stat, _ := m.Compute(chatId, func(oldValue *ChatStat, loaded bool) (newValue *ChatStat, delete bool) {
		if !loaded || oldValue == nil || oldValue.StatDate != day {
			if oldValue != nil {
				// 这里除了忽略错误，还有什么办法呢？
				_ = oldValue.Save(ctx, q)
			}
			daily, err := q.getOrCreateChatStat(ctx, chatId, day)
			if err != nil {
				return nil, true
			}
			return &ChatStat{
				mu:            sync.Mutex{},
				timezone:      timezone,
				ChatStatDaily: daily,
			}, false
		}
		oldValue.timezone = timezone
		return oldValue, false
	})
	return stat, nil
}

func (q *Queries) ChatStatAt(chatId, unixTime int64) *ChatStat {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	cfg, err := q.chatCfgById(ctx, chatId)
	if err != nil {
		return nil
	}
	stat, _ := q.chatStatAtWithTimezone(ctx, chatId, unixTime, cfg.Timezone)
	return stat

}

func (q *Queries) ChatStatToday(chatId int64) (stat *ChatStat) {
	return q.ChatStatAt(chatId, time.Now().Unix())
}
