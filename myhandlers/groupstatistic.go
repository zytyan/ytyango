package myhandlers

import (
	"encoding/gob"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/go-co-op/gocron"
	jsoniter "github.com/json-iterator/go"
	"github.com/kr/pretty"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/rivo/uniseg"
	"html"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"time"
)

type UserMsgStat struct {
	MsgCount int
	MsgLen   int
}
type GroupStatDaily struct {
	MessageCount int64 `json:"message_count"`
	PhotoCount   int64 `json:"photo_count"`
	StickerCount int64 `json:"sticker_count"`
	ForwardCount int64 `json:"forward_count"` // 转发了多少条

	UserMsgStat map[int64]*UserMsgStat `json:"user_msg_stat"`
	// 分时间段计算群内发消息的数量，用于统计频率，每十分钟一个计数
	MsgCountByTime   [24 * 6]int64 `json:"msg_count_by_time"`
	MsgIdAtTimeStart [24 * 6]int64 `json:"msg_id_at_time_start"`

	MarsCount    int64 `json:"mars_count"`     // 火星图的数量，从python处做联动，用httpx直接发过来
	MaxMarsCount int64 `json:"max_mars_count"` // 火星图的最大火星次数

	RacyCount  int64 `json:"racy_count"`  // 色图数量
	AdultCount int64 `json:"adult_count"` // 色图(R18)数量

	DownloadVideoCount int64 `json:"download_video_count"` // 下载视频数量
	DownloadAudioCount int64 `json:"download_audio_count"` // 下载音频数量

	DioAddUserCount int64 `json:"dio_add_user_count"`
	DioBanUserCount int64 `json:"dio_ban_user_count"`
}

var addNewMsgMutex = &sync.Mutex{}

func (g *GroupStatDaily) addNewMsg(msg *gotgbot.Message) {
	addNewMsgMutex.Lock()
	defer addNewMsgMutex.Unlock()
	g.MessageCount++
	if g.UserMsgStat == nil {
		g.UserMsgStat = make(map[int64]*UserMsgStat)
	}
	stat, ok := g.UserMsgStat[msg.From.Id]
	if !ok {
		stat = &UserMsgStat{}
		g.UserMsgStat[msg.From.Id] = stat
	}
	stat.MsgCount++
	stat.MsgLen += uniseg.GraphemeClusterCount(msg.Text)

	// 东八区，每十分钟一个时间段
	msgDate := time.Unix(msg.Date, 0)
	timeSeg := msgDate.Hour()*6 + msgDate.Minute()/10
	g.MsgCountByTime[timeSeg]++
	if g.MsgIdAtTimeStart[timeSeg] == 0 {
		g.MsgIdAtTimeStart[timeSeg] = msg.MessageId
	}
	if msg.ForwardFrom != nil {
		g.ForwardCount++
	}

	switch {
	case msg.Photo != nil:
		g.PhotoCount++
	case msg.Sticker != nil:
		g.StickerCount++
	}
}

// GroupStatTwoDays 由于是8点才发送，所以要保存昨天的计数
type GroupStatTwoDays struct {
	mu         *sync.Mutex
	LastDay    *GroupStatDaily `json:"last_day,omitempty"` // 有可能并不是昨天，而是更早的一天
	Today      *GroupStatDaily `json:"today,omitempty"`
	LastUpdate time.Time       `json:"last_update"`
	NextUpdate time.Time       `json:"next_update"`
}

func (g *GroupStatTwoDays) forceNewDay() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.LastDay = g.Today
	g.Today = &GroupStatDaily{
		UserMsgStat: make(map[int64]*UserMsgStat),
	}
	g.LastUpdate = time.Now()
	// 下次更新为第二天的凌晨四点
	now := time.Now()
	var next4Am time.Time
	if now.Hour() < 4 {
		next4Am = time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, now.Location())
	} else {
		next4Am = time.Date(now.Year(), now.Month(), now.Day()+1, 4, 0, 0, 0, now.Location())
	}
	g.NextUpdate = next4Am
}
func (g *GroupStatTwoDays) tryNewDay() {
	if time.Now().Before(g.NextUpdate) {
		return
	}
	g.forceNewDay()
}

// GetMostActiveTimeSeg 获取群内发言频率最高的时间段
func (g *GroupStatDaily) GetMostActiveTimeSeg() (timeId int, timeSeg string, count int64) {
	timeSegInt := 0
	for i, c := range g.MsgCountByTime {
		if c > count {
			count = c
			timeSegInt = i
			timeId = i
		}
	}
	hour := timeSegInt / 6
	minute := (timeSegInt % 6) * 10
	timeSeg = fmt.Sprintf("%02d:%02d", hour, minute)
	return
}

// var groupStat = make(map[int64]*GroupStatTwoDays)
var groupStat = xsync.NewMapOf[int64, *GroupStatTwoDays]()

// GetMostActiveUsers 获取最活跃的三个用户
func (g *GroupStatDaily) GetMostActiveUsers() (users []int64, counts []int) {
	type userCount struct {
		user int64
		stat *UserMsgStat
	}
	tmp := make([]userCount, 0, len(g.UserMsgStat))
	for u, c := range g.UserMsgStat {
		tmp = append(tmp, userCount{u, c})
	}
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].stat.MsgCount > tmp[j].stat.MsgCount
	})
	for i := 0; i < 3 && i < len(tmp); i++ {
		users = append(users, tmp[i].user)
		counts = append(counts, tmp[i].stat.MsgCount)
	}
	return
}

func (g *GroupStatDaily) String(groupId int64) string {
	if g == nil {
		return "<!nil>没有数据"
	}
	today := time.Now().Format("2006年01月02日")
	actUser, count := g.GetMostActiveUsers()
	timeId, actTime, actTimeCnt := g.GetMostActiveTimeSeg()
	act3Users := make([]string, 0, 3)
	mostActUser := ""
	act3UsersName := ""
	actMaxCnt := 0
	if len(actUser) > 0 {
		actMaxCnt = int(count[0])
	}
	for i := 0; i < 3 && i < len(actUser); i++ {
		act3Users = append(act3Users, GetUser(actUser[i]).Name())
	}
	if len(act3Users) == 0 {
		act3UsersName = "没有人"
		mostActUser = "没有人"
	} else if len(act3Users) == 1 {
		act3UsersName = act3Users[0]
	} else {
		act3UsersName = strings.Join(act3Users[:len(act3Users)-1], "、") + "和" + act3Users[len(act3Users)-1]
	}
	if actMaxCnt > 0 {
		mostActUser = act3Users[0]
	}
	// 转换HTML，避免被tg解析
	act3UsersName = html.EscapeString(act3UsersName)
	mostActUser = html.EscapeString(mostActUser)

	groupLinkId := -groupId - 1000000000000
	link := fmt.Sprintf("https://t.me/c/%d/%d", groupLinkId, g.MsgIdAtTimeStart[timeId])
	tmp := fmt.Sprintf(`早上好！吹水群！
今天是%s，昨天的发言统计，最后的结果是满打满算的整整%d条，你们这些家伙都不用上班的吗？
多亏了%s没完没了的摸鱼吹水，光%s一个人就发了%d条。但有一个晶哥也发话了，我看你们全都得喝茶，因为平子肯定咽不下这口气。
群里一共发了%d张图片，还有%d个表情，又是只发表情的社恐干的好事。与此同时，毅力号还在火星上替火星人找到了%d张图，火星次数最多的图让你们火星了%d次，真是一群火星人。
群里发了%d张色图，里面还有%d张R18，你们今天发色图，明天FBI就来敲你家门了。
智乃酱帮你们下了%d个视频，%d个音频，今日份的娱乐就到这里吧。
而群里最热闹的<a href="%s">%s</a>，这十分钟居然发了%d条，好吧，吹水群还是那个吹水群。
我是你们的铁哥们智乃酱，和我一起开启完蛋操的新一天吧！`,
		today, g.MessageCount,
		act3UsersName, mostActUser, actMaxCnt,
		g.PhotoCount, g.StickerCount, g.MarsCount, g.MaxMarsCount,
		g.AdultCount+g.RacyCount, g.AdultCount,
		g.DownloadVideoCount, g.DownloadAudioCount,
		link, actTime, actTimeCnt)
	return tmp
}
func (g *GroupStatDaily) GetRank() string {
	type userCount struct {
		user  int64
		count int64
	}
	tmp := make([]userCount, 0, len(g.UserMsgStat))
	for u, c := range g.UserMsgStat {
		tmp = append(tmp, userCount{u, int64(c.MsgCount)})
	}
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].count > tmp[j].count
	})
	res := make([]string, 0, len(tmp))
	for i, v := range tmp {
		if i >= 10 {
			break
		}
		res = append(res, fmt.Sprintf("%s: %d", GetUser(v.user).Name(), v.count))
	}
	if len(res) > 10 {
		sum := 0
		for i := 10; i < len(tmp); i++ {
			sum += int(tmp[i].count)
		}
		res = append(res, fmt.Sprintf("其他人: %d", sum))
	}
	return strings.Join(res, "\n")
}

func GetRank(bot *gotgbot.Bot, ctx *ext.Context) error {
	var text string
	WithGroupLockToday(ctx.EffectiveChat.Id, func(daily *GroupStatDaily) {
		text = daily.GetRank()
	})
	if text == "" {
		text = "没有数据"
	}
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, text, nil)
	return err
}

func GetCntByTime(bot *gotgbot.Bot, ctx *ext.Context) error {
	var text string
	var textB []byte
	var err error
	WithGroupLockToday(ctx.EffectiveChat.Id, func(daily *GroupStatDaily) {
		textB, err = jsoniter.Marshal(daily.MsgCountByTime)
		if err != nil {
			return
		}
		text = string(textB)
	})
	if err != nil {
		text = err.Error()
	} else if text == "" {
		text = "没有数据"
	}
	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, nil)
	return err
}

func AddNewMsg(_ *gotgbot.Bot, ctx *ext.Context) error {
	stat, _ := groupStat.LoadOrCompute(ctx.Message.Chat.Id, func() *GroupStatTwoDays {
		return &GroupStatTwoDays{mu: &sync.Mutex{}, Today: &GroupStatDaily{}, LastUpdate: time.Now()}
	})
	stat.tryNewDay()
	stat.Today.addNewMsg(ctx.Message)
	return nil
}

func sendGroupStat(bot *gotgbot.Bot, groupId int64, isToday bool) error {
	stat, ok := groupStat.Load(groupId)
	if !ok {
		_, err := bot.SendMessage(groupId, "错误：这个群没有开启这个功能", nil)
		return err
	}
	stat.tryNewDay()
	var text string
	if isToday {
		text = stat.Today.String(groupId)
	} else {
		text = stat.LastDay.String(groupId)
	}
	_, err := bot.SendMessage(groupId, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}
func GroupStatDiagnostic(bot *gotgbot.Bot, ctx *ext.Context) error {
	filename := fmt.Sprintf("groupstat_%d.txt", ctx.EffectiveChat.Id)
	t, err := os.Create(filename)
	if err != nil {
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, "创建文件错误", nil)
		return err
	}
	defer t.Close()
	defer os.Remove(filename)
	_, _ = t.WriteString(fmt.Sprintf("run count: %d, next run: %s\n", job.RunCount(), job.NextRun().Format("2006-01-02 15:04:05")))
	_, _ = t.WriteString(fmt.Sprintf("group stat count: %d\n", groupStat.Size()))
	groupStat.Range(func(k int64, v *GroupStatTwoDays) bool {
		_, _ = t.WriteString(fmt.Sprintf("group %d, last update: %s, next update: %s\n", k, v.LastUpdate.Format("2006-01-02 15:04:05"), v.NextUpdate.Format("2006-01-02 15:04:05")))
		_, _ = t.WriteString(fmt.Sprintf("today message count: %d\n", v.Today.MessageCount))
		_, _ = t.WriteString(fmt.Sprintf("last day message count: %d\n", v.LastDay.MessageCount))
		_, _ = t.WriteString(pretty.Sprintf("today: %# v\n", v.Today))
		_, _ = t.WriteString(pretty.Sprintf("last day: %# v\n", v.LastDay))
		return true
	})
	_, err = bot.SendDocument(ctx.EffectiveChat.Id, fileSchema(filename), nil)
	return err
}

func SendGroupStat(bot *gotgbot.Bot, ctx *ext.Context) error {
	err := sendGroupStat(bot, ctx.Message.Chat.Id, false)
	if err != nil {
		return err
	}
	err = sendGroupStat(bot, ctx.Message.Chat.Id, true)
	return err
}

func ForceNewDay(bot *gotgbot.Bot, ctx *ext.Context) error {
	stat, _ := groupStat.LoadOrCompute(ctx.Message.Chat.Id, func() *GroupStatTwoDays {
		return &GroupStatTwoDays{mu: &sync.Mutex{}, Today: &GroupStatDaily{}, LastUpdate: time.Now()}
	})
	stat.forceNewDay()
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, "强制新的一天~", nil)
	return err
}

func WithGroupLockToday(groupId int64, f func(daily *GroupStatDaily)) {
	stat, ok := groupStat.Load(groupId)
	if !ok {
		return
	}
	stat.mu.Lock()
	defer stat.mu.Unlock()
	if stat.Today == nil {
		log.Errorf("group %d today is nil", groupId)
	}
	f(stat.Today)
}

func AddDownloadVideoCnt(chatId int64) {
	WithGroupLockToday(chatId, func(daily *GroupStatDaily) {
		daily.DownloadVideoCount++
	})
}
func AddDownloadAudioCnt(chatId int64) {
	WithGroupLockToday(chatId, func(daily *GroupStatDaily) {
		daily.DownloadAudioCount++
	})
}

func GetGroupStat(groupId int64) (*GroupStatTwoDays, bool) {
	return groupStat.Load(groupId)
}

var sendScheduler *gocron.Scheduler
var job *gocron.Job

const statFile = "groupstat.gob"
const statJsonFile = "groupstat.json"

var loadSuccess = false

func saveStat() {
	if !loadSuccess {
		return
	}
	var tmpGroupStat = make(map[int64]*GroupStatTwoDays)
	groupStat.Range(func(k int64, v *GroupStatTwoDays) bool {
		tmpGroupStat[k] = v
		return true
	})
	fb, err := os.Create(statFile)
	if err != nil {
		log.Errorf("save stat failed %s", err)
		return
	}
	defer fb.Close()
	e := gob.NewEncoder(fb)
	err = e.Encode(tmpGroupStat)
	if err != nil {
		log.Errorf("save stat failed %s", err)
		return
	}
	log.Infof("save stat success")
}

func loadStat() {
	fb, err := os.Open(statFile)
	if err != nil {
		// 文件没有就不怕覆盖了
		loadSuccess = true
		log.Errorf("load stat failed %s", err)
		return
	}
	var tmpGroupStat = make(map[int64]*GroupStatTwoDays)
	d := gob.NewDecoder(fb)
	err = d.Decode(&tmpGroupStat)
	if err != nil {
		log.Errorf("load stat failed %s", err)
		return
	}
	for _, v := range tmpGroupStat {
		v.mu = &sync.Mutex{}
	}
	groupStat = xsync.NewMapOf[int64, *GroupStatTwoDays]()
	for k, v := range tmpGroupStat {
		groupStat.Store(k, v)
	}
	loadSuccess = true
	log.Infof("load stat success")
}

func init() {
	loadStat()
	chn := make(chan os.Signal, 1)
	signal.Notify(chn, os.Interrupt)
	go func() {
		<-chn
		saveStat()
		os.Exit(0)
	}()
	sendScheduler = gocron.NewScheduler(time.Local)
	var err error
	job, err = sendScheduler.Every(1).Day().At("08:00").Do(func() {
		log.Info("send group stat")
		err := sendGroupStat(GetMainBot(), -1001471592463, false)
		if err != nil {
			log.Errorf("send group stat failed %s", err)
		}
	})
	sendScheduler.StartAsync()
	if err != nil {
		panic(err)
	}
}
