package myhandlers

import (
	"errors"
	"gorm.io/gorm"
	"main/globalcfg"
	"main/helpers/azure"
	"sync"
)

type ModeratorConfig struct {
	AdultThreshold float64 `gorm:"default:0.7"`
	RacyThreshold  float64 `gorm:"default:0.7"`
}

type GroupInfo struct {
	ID         int64 `gorm:"primaryKey"`
	GroupID    int64 `gorm:"unique"`
	GroupWebId int64 `gorm:"index"`

	AutoCvtBili     bool
	AutoOcr         bool
	AutoCalculate   bool
	AutoExchange    bool
	ParseFlags      bool // 是否要解析群友立下的flag
	AutoCheckAdult  bool
	CoCEnabled      bool
	ModeratorConfig ModeratorConfig `gorm:"embedded;embeddedPrefix:moderator_"`

	SaveMessages bool
}

var groupInfoCache = mustNewLru[int64, *GroupInfo](1000)
var groupInfoWMutex = sync.Mutex{}

func GetGroupInfo(groupId int64) *GroupInfo {
	info, err := getGroupInfo(groupId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newInfo := &GroupInfo{
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
	var info GroupInfo
	if res, found := groupInfoCache.Get(groupId); found {
		log.Debugf("get group info %d from groupInfoCache", groupId)
		return res, nil
	}
	res := globalcfg.GetDb().Where("group_id = ?", groupId).Take(&info)
	if res.Error != nil {
		return nil, res.Error
	}
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

func (g *GroupInfo) Update() error {
	db := globalcfg.GetDb()
	db.Model(g).Updates(map[string]any{
		"auto_convert_bilibili": g.AutoCvtBili,
		"auto_ocr":              g.AutoOcr,
		"auto_check_adult":      g.AutoCheckAdult,
		"auto_calculate":        g.AutoCalculate,
		"auto_exchange":         g.AutoExchange,
		"parse_flags":           g.ParseFlags,
		"save_messages":         g.SaveMessages,
	})
	groupInfoCache.Add(g.GroupID, g)
	return nil
}

func (g *GroupInfo) UpdateWebId(newId int64) error {
	db := globalcfg.GetDb()
	db.Model(g).Updates(map[string]any{
		"group_web_id": newId,
	})
	groupInfoCache.Add(g.GroupID, g)
	return nil
}

func (g *GroupInfo) HasWebId() bool {
	return g.GroupWebId != 0
}

func (m *ModeratorConfig) IsAdult(res *azure.ModeratorResult) bool {
	return res.AdultClassificationScore > m.AdultThreshold
}

func (m *ModeratorConfig) IsRacy(res *azure.ModeratorResult) bool {
	return res.RacyClassificationScore > m.RacyThreshold
}
