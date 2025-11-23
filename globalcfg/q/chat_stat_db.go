package q

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/gob"
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
	return gob.NewDecoder(bytes.NewReader(data)).Decode(t)
}

func (t TenMinuteStats) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(t)
	return buf.Bytes(), err
}

type ChatStat struct {
	mu sync.Mutex
	ChatStatDaily
}

func (s *ChatStat) IncMessage(userId, txtLen, time int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.MessageCount++
	s.MsgCountByTime[(time%86400)/(24*10)]++
	s.UserMsgStat[userId].MsgCount++
	s.UserMsgStat[userId].MsgLen += txtLen
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
	if err != nil {
		return daily, err
	}
	return q.createChatStatDaily(ctx, chatId, day)
}

func (q *Queries) GetChatStatToday(chatId int64, day int64) (stat *ChatStat) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	var err error
	stat, _ = m.Compute(chatId, func(oldValue *ChatStat, loaded bool) (newValue *ChatStat, delete bool) {
		if !loaded || oldValue == nil || oldValue.StatDate != day {
			if oldValue.StatDate != day {
				_ = oldValue.Save(ctx, q) // 忽略错误，不忽略也不知道怎么办
			}
			var daily ChatStatDaily
			daily, err = q.getOrCreateChatStat(ctx, chatId, day)
			if err != nil {
				return nil, true
			}
			return &ChatStat{
				mu:            sync.Mutex{},
				ChatStatDaily: daily,
			}, false
		}
		return oldValue, false
	})
	return
}
