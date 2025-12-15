package q

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/gob"
	"fmt"
	"main/helpers/lrusf"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	daySeconds int64 = 24 * 60 * 60
	tenMinutes       = 10 * 60
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

type ChatStatKey struct {
	Day int64
	Id  int64
}

var chatStatCache *lrusf.Cache[ChatStatKey, *ChatStat]

// FlushChatStats saves all in-memory chat statistics to the database.
// It is safe to call concurrently with other stat operations.
func (q *Queries) FlushChatStats(ctx context.Context) error {
	var firstErr error
	q.logger.Info("flushing chat_stats")
	for _, stat := range chatStatCache.Range() {
		q.logger.Info("flushing", zap.Any("stat", stat))
		if stat == nil {
			continue
		}
		if err := stat.Save(ctx, q); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (q *Queries) getOrCreateChatStat(ctx context.Context, chatId int64, day int64) (ChatStatDaily, error) {
	daily, err := q.getChatStat(ctx, chatId, day)
	if err != nil {
		return q.createChatStatDaily(ctx, chatId, day)
	}
	return daily, err

}

func (q *Queries) chatStatAtWithTimezone(ctx context.Context, chatId, unixTime, timezone int64) (*ChatStat, error) {
	day := (unixTime + timezone) / daySeconds
	key := ChatStatKey{
		Day: day,
		Id:  chatId,
	}
	return chatStatCache.Get(key, func() (*ChatStat, error) {
		daily, err := q.getOrCreateChatStat(ctx, chatId, unixTime)
		if err != nil {
			return nil, err
		}
		stat := &ChatStat{
			mu:            sync.Mutex{},
			timezone:      timezone,
			ChatStatDaily: daily}
		return stat, nil
	})
}

func (q *Queries) ChatStatAt(chatId, unixTime int64) *ChatStat {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	timezone := q.GetChatCfgByIdOrDefault(chatId).Timezone
	stat, _ := q.chatStatAtWithTimezone(ctx, chatId, unixTime, timezone)
	return stat
}

func (q *Queries) ChatStatNow(chatId int64) (stat *ChatStat) {
	return q.ChatStatAt(chatId, time.Now().Unix())
}

// ChatStatOfDay returns the stat of the day which contains the unixTime in the chat's timezone.
func (q *Queries) ChatStatOfDay(ctx context.Context, chatId, unixTime int64) (ChatStatDaily, int64, error) {
	cfg, err := q.GetChatCfgById(ctx, chatId)
	if err != nil {
		return ChatStatDaily{}, 0, err
	}
	daily, err := q.chatStatAtWithTimezone(ctx, chatId, unixTime, cfg.Timezone)
	if err != nil {
		return ChatStatDaily{}, 0, err
	}
	return daily.ChatStatDaily, cfg.Timezone, err
}
