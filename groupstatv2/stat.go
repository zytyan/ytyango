package groupstatv2

import (
	"encoding/gob"
	"os"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rivo/uniseg"
)

type UserMsgStat struct {
	MsgCount Counter
	MsgLen   Counter
}

type GroupStatDaily struct {
	MessageCount Counter
	PhotoCount   Counter
	VideoCount   Counter
	StickerCount Counter
	ForwardCount Counter // 转发了多少条

	UserMsgStat *EncodableMap[int64, *UserMsgStat]
	// 分时间段计算群内发消息的数量，用于统计频率，每十分钟一个计数
	MsgCountByTime   [24 * 6]Counter
	MsgIdAtTimeStart [24 * 6]Counter

	MarsCount    Counter // 火星图的数量，从python处做联动，用httpx直接发过来
	MaxMarsCount Counter // 火星图的最大火星次数

	RacyCount  Counter // 色图数量
	AdultCount Counter // 色图(R18)数量

	DownloadVideoCount Counter // 下载视频数量
	DownloadAudioCount Counter // 下载音频数量

	DioAddUserCount Counter
	DioBanUserCount Counter
}

type GroupStatTwoDays struct {
	mu         sync.Mutex
	Yesterday  *GroupStatDaily // 有可能并不是昨天，而是更早的一天
	Today      *GroupStatDaily
	LastUpdate time.Time
	NextUpdate time.Time
	TimeOffset int64 // 使用比较基础的秒来做偏移量
}

func (g *GroupStatTwoDays) forceNewDay() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Yesterday = g.Today
	g.Today = &GroupStatDaily{
		// 一个群里有64个人发言都很神奇了
		UserMsgStat: NewEncodableMapOf[int64, *UserMsgStat](64),
	}
	// 下次更新为第二天的凌晨四点
	now := time.Now()
	var next4Am time.Time
	if now.Hour() < 4 {
		next4Am = time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, now.Location())
	} else {
		next4Am = time.Date(now.Year(), now.Month(), now.Day()+1, 4, 0, 0, 0, now.Location())
	}
	g.LastUpdate = now
	g.NextUpdate = next4Am
}

func (g *GroupStatTwoDays) tryNewDay() {
	if time.Now().Before(g.NextUpdate) {
		return
	}
	g.forceNewDay()
}

func (g *GroupStatTwoDays) AddMsg(msg *gotgbot.Message) {
	g.tryNewDay()
	if msg == nil {
		return
	}
	stat := g.Today
	stat.MessageCount.Inc()
	if len(msg.Photo) != 0 {
		stat.PhotoCount.Inc()
	}
	if msg.Video != nil {
		stat.VideoCount.Inc()
	}
	if msg.Sticker != nil {
		stat.StickerCount.Inc()
	}
	if msg.ForwardOrigin != nil {
		stat.ForwardCount.Inc()
	}
	idx := ((msg.Date + g.TimeOffset) % (24 * 60 * 60)) / (10 * 60)
	stat.MsgCountByTime[idx].Inc()
	stat.MsgCountByTime[idx].CompareAndSwap(0, msg.MessageId)
	userStat, _ := stat.UserMsgStat.LoadOrCompute(1, func() *UserMsgStat {
		return &UserMsgStat{}
	})
	length := uniseg.GraphemeClusterCount(msg.Text)
	userStat.MsgCount.Inc()
	userStat.MsgLen.Add(int64(length))
}

func newGroupStatTwoDays() *GroupStatTwoDays {
	res := &GroupStatTwoDays{}
	res.forceNewDay()
	return res
}

const statFile = "groupstat_v2.gob"

var groupStat EncodableMap[int64, *GroupStatTwoDays]
var globalMu sync.Mutex

func checkGroupStat() {
	if groupStat.MapOf != nil {
		return
	}
	globalMu.Lock()
	defer globalMu.Unlock()
	if groupStat.MapOf != nil {
		return
	}
	groupStat = *NewEncodableMapOf[int64, *GroupStatTwoDays](64)
}

func GetGroup(id int64) *GroupStatTwoDays {
	checkGroupStat()
	g, _ := groupStat.LoadOrCompute(id, newGroupStatTwoDays)
	return g
}

func GetGroupToday(id int64) *GroupStatDaily {
	return GetGroup(id).Today
}

func AddMsg(msg *gotgbot.Message) {
	GetGroup(msg.Chat.Id).AddMsg(msg)
}

func SaveToFile() error {
	globalMu.Lock()
	defer globalMu.Unlock()
	tmpFile := statFile + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer f.Close()
	err = gob.NewEncoder(f).Encode(&groupStat)
	if err != nil {
		return err
	}
	err = os.Rename(tmpFile, statFile)
	return err
}

func LoadFromFile() error {
	globalMu.Lock()
	defer globalMu.Unlock()
	f, err := os.Open(statFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewDecoder(f).Decode(&groupStat)
}
