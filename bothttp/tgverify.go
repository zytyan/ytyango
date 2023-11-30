package bothttp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type WebInitUser struct {
	Id              int    `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPm bool   `json:"allows_write_to_pm"`
}

type AuthInfo struct {
	QueryId  string      `json:"query_id"`
	User     WebInitUser `json:"user"`
	AuthDate time.Time   `json:"auth_date"`
	Hash     string      `json:"hash"`
}

func checkTelegramAuth(str string, verifyKey []byte) (res AuthInfo, err error) {
	split := strings.Split(str, "&")
	const hashPrefix = "hash"
	recvHash := ""
	data := make([]string, 0, len(split))
	for _, v := range split {
		key, value, _ := strings.Cut(v, "=")
		if key == hashPrefix {
			recvHash = value
			continue
		}
		key, err1 := url.QueryUnescape(key)
		value, err2 := url.QueryUnescape(value)
		if err1 != nil || err2 != nil {
			err = fmt.Errorf("url unescape err %v %v", err1, err2)
			return
		}
		data = append(data, key+"="+value)
	}
	if recvHash == "" {
		err = fmt.Errorf("no hash")
		return
	}

	slices.Sort(data)
	initData := []byte(strings.Join(data, "\n"))
	mac := hmac.New(sha256.New, verifyKey)
	mac.Write(initData)
	calcHash := hex.EncodeToString(mac.Sum(nil))
	if recvHash != calcHash {
		err = fmt.Errorf("wrong recvHash calc=%s*** recv=%s", calcHash[:4], recvHash)
		return
	}
	for _, v := range data {
		key, value, _ := strings.Cut(v, "=")
		switch key {
		case "auth_date":
			parseInt, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return AuthInfo{}, err
			}
			res.AuthDate = time.Unix(parseInt, 0)
		case "hash":
			res.Hash = value
		case "query_id":
			res.QueryId = value
		case "user":
			var user WebInitUser
			err = jsoniter.Unmarshal([]byte(value), &user)
			if err != nil {
				return
			}
			res.User = user
		}
	}
	return
}

func verifyHeader(ctx *gin.Context) {
	if devModeCheck(ctx) {
		return
	}
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.AbortWithStatusJSON(401, ErrNoAuth.Msg("未提供验证信息"))
		return
	}
	var data string
	const TelegramPrefix = "Telegram "
	if strings.HasPrefix(authHeader, TelegramPrefix) {
		data = authHeader[len(TelegramPrefix):]
	} else {
		ctx.AbortWithStatusJSON(401, ErrValidFailed.Msg("暂不支持非Telegram验证方式"))
		return
	}
	auth, err := checkTelegramAuth(data, botVerifyKey)
	if err != nil {
		ctx.AbortWithStatusJSON(401, ErrValidFailed.Msg("验证用于身份失败"+err.Error()))
		return
	}
	if time.Now().Sub(auth.AuthDate) > 4*time.Hour {
		ctx.AbortWithStatusJSON(401, ErrExpired.Msg("数据过期，该网页验证时长已超过4小时，需要重新打开网页验证"))
		return
	}
	ctx.Set("auth", auth)
	ctx.Next()
	return
}
