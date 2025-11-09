package bothttp

import (
	"crypto/hmac"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTelegramAuth(t *testing.T) {
	as := require.New(t)
	key := hmac.New(sha256.New, []byte("WebAppData"))
	// 该token已经失效，可放心在测试代码中使用
	key.Write([]byte("554277510:AAEKxRdcRfhEjtSIfxpaYtL19XFgdDcY23U"))
	keyS := key.Sum(nil)
	data := "query_id=AAGhdeMLAAAAAKF14wu7BOmF&user=%7B%22" +
		"id%22%3A199456161%2C%22first_name%22%3A%22z%22%2C" +
		"%22last_name%22%3A%22%22%2C%22username%22%3A%22Yt" +
		"yan%22%2C%22language_code%22%3A%22zh-hans%22%2C%2" +
		"2is_premium%22%3Atrue%2C%22allows_write_to_pm%22%3" +
		"Atrue%7D&auth_date=1695737566&hash=0cd9a1d70d6b6c" +
		"d83630e455eb72d89dfd955a5068272c621402684358f32d68"
	// 字符串长度：341
	auth, err := checkTelegramAuth(data, keyS)
	as.NoError(err)
	as.NotEmptyf(auth, "auth is empty")
}
