package myhandlers

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"main/globalcfg"
	"main/helpers/azure"
	"reflect"
	"sync"
	"time"
)

type ModeratorConfig struct {
	AdultThreshold float64 `gorm:"default:0.7"`
	RacyThreshold  float64 `gorm:"default:0.7"`
}

type GroupInfo struct {
	mu         *sync.Mutex
	timer      *time.Timer
	ID         int64 `gorm:"primaryKey"`
	GroupID    int64 `gorm:"unique"`
	GroupWebId int64 `gorm:"index"`

	AutoCvtBili     bool `btnTxt:"自动转换Bilibili视频链接" pos:"1,1"`
	AutoOcr         bool
	AutoCalculate   bool `btnTxt:"自动计算算式" pos:"2,1"`
	AutoExchange    bool `btnTxt:"自动换算汇率" pos:"2,2"`
	ParseFlags      bool
	AutoCheckAdult  bool
	CoCEnabled      bool            `btnTxt:"启用CoC辅助" pos:"3,2"`
	ModeratorConfig ModeratorConfig `gorm:"embedded;embeddedPrefix:moderator_"`

	SaveMessages bool `btnTxt:"保存群组消息" pos:"3,1"`
}

var groupInfoCache = mustNewLru[int64, *GroupInfo](1000)
var groupInfoWMutex = sync.Mutex{}

func GetGroupInfo(groupId int64) *GroupInfo {
	info, err := getGroupInfo(groupId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newInfo := &GroupInfo{
				mu:         &sync.Mutex{},
				GroupID:    groupId,
				GroupWebId: 0,

				AutoCvtBili:    false,
				AutoOcr:        false,
				AutoCalculate:  false,
				AutoExchange:   false,
				ParseFlags:     false,
				AutoCheckAdult: false,

				SaveMessages: true,
			}
			CreateGroupInfo(newInfo)
			return newInfo
		}
		log.Warnf("get group info %d error %s", groupId, err)
		return nil
	}
	return info
}

func getGroupInfo(groupId int64) (*GroupInfo, error) {
	groupInfoWMutex.Lock()
	defer groupInfoWMutex.Unlock()
	var info GroupInfo
	if res, found := groupInfoCache.Get(groupId); found {
		res.mu = &sync.Mutex{}
		log.Debugf("get group info %d from groupInfoCache", groupId)
		return res, nil
	}
	res := globalcfg.GetDb().Where("group_id = ?", groupId).Take(&info)
	if res.Error != nil {
		return nil, res.Error
	}
	info.mu = &sync.Mutex{}
	groupInfoCache.Add(groupId, &info)
	return &info, nil
}

func GetGroupInfoUseWebId(webId int64) *GroupInfo {
	var info GroupInfo
	res := globalcfg.GetDb().Where("group_web_id = ?", webId).Take(&info)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		log.Warnf("get group info %d error %s", webId, res.Error)
		return nil
	}
	groupInfoCache.Add(info.GroupID, &info)
	return &info
}

func CreateGroupInfo(info *GroupInfo) {
	groupInfoWMutex.Lock()
	defer groupInfoWMutex.Unlock()
	_, err := getGroupInfo(info.GroupID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		globalcfg.GetDb().Create(info)
		groupInfoCache.Add(info.GroupID, info)
		return
	}
	if err == nil {
		log.Debugf("group info %d already exists", info.GroupID)
		return
	}
}

func (g *GroupInfo) UpdateNow() error {
	log.Infof("update group info %d", g.ID)
	return globalcfg.GetDb().Save(g).Error
}

func (g *GroupInfo) Update() {
	log.Infof("update group info %d after 3 seconds", g.ID)
	if g.timer != nil {
		g.timer.Reset(3 * time.Second)
		return
	}
	g.timer = time.AfterFunc(3*time.Second, func() {
		err := g.UpdateNow()
		if err != nil {
			log.Warnf("update group info %d error %s", g.GroupID, err)
		}
	})
	return
}

// SetFieldByName 仅允许修改带 btnTxt 标签的字段
func (g *GroupInfo) SetFieldByName(fieldName string, value any) error {
	v := reflect.ValueOf(g).Elem()
	t := v.Type()

	f, ok := t.FieldByName(fieldName)
	if !ok {
		return fmt.Errorf("no such field: %s", fieldName)
	}

	tag := f.Tag.Get("btnTxt")
	if tag == "" {
		return fmt.Errorf("field %s has no btnTxt tag (not allowed to modify)", fieldName)
	}
	fieldVal := v.FieldByName(fieldName)
	if !fieldVal.CanSet() {
		return fmt.Errorf("cannot set field: %s", fieldName)
	}

	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Bool {
		return fmt.Errorf("invalid value type: expected bool, got %v", val.Kind())
	}

	fieldVal.SetBool(val.Bool())
	return nil
}

type BtnField struct {
	Name     string // 字段名
	Text     string // btnTxt 内容
	Value    bool   // 字段值
	Position [2]int // 所在位置
}

// GetBtnTxtFields 返回所有带有 btnTxt 且类型为 bool 的字段（按定义顺序）
func (g *GroupInfo) GetBtnTxtFields() []BtnField {
	v := reflect.ValueOf(g).Elem()
	t := v.Type()

	var fields []BtnField
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("btnTxt")
		posTag := field.Tag.Get("pos")
		if tag != "" && posTag != "" && field.Type.Kind() == reflect.Bool {
			var row, col int
			_, _ = fmt.Sscanf(posTag, "%d,%d", &row, &col)
			fields = append(fields, BtnField{
				Name:     field.Name,
				Text:     tag,
				Value:    v.Field(i).Bool(),
				Position: [2]int{row, col},
			})
		}
	}
	return fields
}
func (g *GroupInfo) GetBtnTxtFieldByName(name string) BtnField {
	v := reflect.ValueOf(g).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("btnTxt")
		posTag := field.Tag.Get("pos")
		if tag != "" && field.Name == name && field.Type.Kind() == reflect.Bool {
			var row, col int
			_, _ = fmt.Sscanf(posTag, "%d,%d", &row, &col)

			return BtnField{
				Name:     field.Name,
				Text:     tag,
				Value:    v.Field(i).Bool(),
				Position: [2]int{row, col},
			}
		}
	}
	return BtnField{Name: "???"}
}
func (m *ModeratorConfig) IsAdult(res *azure.ModeratorResult) bool {
	return res.AdultClassificationScore > m.AdultThreshold
}

func (m *ModeratorConfig) IsRacy(res *azure.ModeratorResult) bool {
	return res.RacyClassificationScore > m.RacyThreshold
}
