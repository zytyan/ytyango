package myhandlers

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	lru "github.com/hashicorp/golang-lru/v2"
	"gorm.io/gorm"
	"main/globalcfg"
	"strings"
	"time"
)

type User struct {
	ID        uint  `gorm:"primaryKey" json:"-" json:"-"`
	UpdatedAt int64 `gorm:"autoUpdateTime" json:"-"`

	UserId    int64         `gorm:"unique" json:"user_id,omitempty"`
	FirstName string        `json:"first_name,omitempty"`
	LastName  string        `json:"last_name"`
	TimeZone  sql.NullInt32 `json:"time_zone" gorm:"default:null"`

	ProfileUpdateAt int64  `json:"-"`
	ProfilePhoto    string `json:"-"`
}

func mustNewLru[K comparable, V any](size int) *lru.Cache[K, V] {
	a, err := lru.New[K, V](size)
	if err != nil {
		panic(err)
	}
	return a
}

var userCache = mustNewLru[int64, *User](1000)

func (u *User) Name() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

func (u *User) GetUpdateTime() time.Time {
	return time.Unix(u.UpdatedAt, 0)
}

func (u *User) needUpdate() bool {
	// 这里主要避免 user profile 被频繁更新
	return time.Since(u.GetUpdateTime()) > time.Hour
}

func (u *User) profileNeedUpdate() bool {
	// 这里会多一次 http bot api 调用，所以冷静一下
	return time.Since(time.Unix(u.ProfileUpdateAt, 0)) > time.Hour*24*3
}

func (u *User) updateName(firstName, lastName string) error {
	if u.FirstName == firstName && u.LastName == lastName {
		return nil
	}
	u.FirstName = firstName
	u.LastName = lastName
	return globalcfg.GetDb().Model(u).
		Select("first_name", "last_name").
		Updates(u).Error
}

func (u *User) updateProfilePhoto(photo string) error {
	if photo == "" || u.ProfilePhoto == photo {
		return nil
	}
	u.ProfilePhoto = photo
	return globalcfg.GetDb().Model(u).
		Select("profile_photo").
		Updates(u).Error
}
func (u *User) UpdateProfilePhoto(bot *gotgbot.Bot) error {
	if !u.profileNeedUpdate() {
		return nil
	}
	profilePhoto, err := bot.GetUserProfilePhotos(u.UserId, nil)
	if err != nil {
		return err
	}
	if len(profilePhoto.Photos) == 0 {
		return nil
	}
	photo := profilePhoto.Photos[0][len(profilePhoto.Photos[0])-1].FileId
	return u.updateProfilePhoto(photo)
}
func (u *User) DownloadProfilePhoto(bot *gotgbot.Bot) (string, error) {
	if u.ProfilePhoto == "" {
		return "", errors.New("no profile photo")
	}
	file, err := bot.GetFile(u.ProfilePhoto, nil)
	if err != nil {
		return file.FilePath, err
	}
	return file.FilePath, err
}
func GetUser(userId int64) *User {
	if user, found := userCache.Get(userId); found {
		return user
	}
	var user User
	res := globalcfg.GetDb().Where("user_id = ?", userId).Take(&user) // 使用take，而不是first，first会有不必要的order by
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Warnf("获取用户信息失败：%s", res.Error.Error())
		}
		return nil
	}
	return &user
}

func botGetUserProfilePhotoFileId(bot *gotgbot.Bot, ctx *ext.Context) (string, error) {
	if ctx.EffectiveUser == nil {
		return "", nil
	}
	photo, err := ctx.EffectiveUser.GetProfilePhotos(bot, nil)
	if err != nil {
		return "", err
	}
	if len(photo.Photos) == 0 {
		return "", nil
	}
	profilePhoto := photo.Photos[0][len(photo.Photos[0])-1].FileId
	return profilePhoto, nil
}

func SaveUser(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveUser == nil {
		return nil
	}
	var err error
	if user := GetUser(ctx.EffectiveUser.Id); user != nil {
		err = user.updateName(ctx.EffectiveUser.FirstName, ctx.EffectiveUser.LastName)
		if err != nil {
			err = fmt.Errorf("更新用户信息失败：%w", err)
		}
		if user.profileNeedUpdate() {
			var profilePhoto string
			profilePhoto, err = botGetUserProfilePhotoFileId(bot, ctx)
			if err != nil {
				err = fmt.Errorf("获取用户头像失败：%w", err)
			}
			err = user.updateProfilePhoto(profilePhoto)
			if err != nil {
				err = fmt.Errorf("更新用户头像失败：%w", err)
			}
		}
		return err

	} else {
		var profilePhoto string
		profilePhoto, err = botGetUserProfilePhotoFileId(bot, ctx)
		if err != nil {
			err = fmt.Errorf("获取用户头像失败：%w", err)
		}
		newUser := &User{
			UserId:          ctx.EffectiveUser.Id,
			FirstName:       ctx.EffectiveUser.FirstName,
			LastName:        ctx.EffectiveUser.LastName,
			ProfileUpdateAt: time.Now().Unix(),
			ProfilePhoto:    profilePhoto,
		}
		err = globalcfg.GetDb().Create(newUser).Error
		if err != nil {
			return fmt.Errorf("创建用户失败：%w", err)
		}
		userCache.Add(newUser.UserId, newUser)
		return nil
	}
}
