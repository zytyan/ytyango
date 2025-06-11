package globalcfg

import (
	"encoding/json"
	"fmt"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
	"testing"
)

/*
bot-token: "***REMOVED***"

god: 199456161
my-chats:
  - -1001471592463
  - -1001126241898
  - -1001170816274

mongo-url: ***REMOVED***

content-Moderator:
  endpoint: "https://sese-detect.cognitiveservices.azure.com"
  api-key: "***REMOVED***"
Ocr:
  endpoint: "https://bot-cv.cognitiveservices.azure.com"
  api-key: '***REMOVED***'
  api-ver: "2023-02-01-preview"
  language: ""
  features: ""

qr-scan-url: ***REMOVED***
save-message: true
tg-api-url: ***REMOVED***
drop-pending-updates: false

meili-config:
  base-url: ***REMOVED***
  index-name: tgmsgs
  primary-key: mongo_id
  master-key:

sese:
  adult-threshold: 0.8
  racy-threshold: 0.6

log-level: -1

*/
// test
func TestGetConfig(t *testing.T) {
	as := require.New(t)
	config := GetConfig()
	as.True(config != nil)
	as.Equal(config.BotToken, "***REMOVED***")
	as.Equal(config.God, int64(199456161))
	as.Equal(config.ContentModerator.Endpoint, "https://sese-detect.cognitiveservices.azure.com")
	as.Equal(config.ContentModerator.ApiKey, "***REMOVED***")
	as.Equal(config.Ocr.Endpoint, "https://bot-cv.cognitiveservices.azure.com")
	as.Equal(config.Ocr.ApiKey, "***REMOVED***")
	as.Equal(config.Ocr.ApiVer, "2023-02-01-preview")
	as.Equal(config.Ocr.Language, "")
	as.Equal(config.Ocr.Features, "")
	as.Equal(config.QrScanUrl, "***REMOVED***")
	as.Equal(config.SaveMessage, true)
	as.Equal(config.TgApiUrl, "***REMOVED***")
	as.Equal(config.DropPendingUpdates, false)
	as.Equal(config.MeiliConfig.BaseUrl, "***REMOVED***")
	as.Equal(config.MeiliConfig.IndexName, "tgmsgs")
	as.Equal(config.MeiliConfig.PrimaryKey, "mongo_id")
	as.Equal(config.MeiliConfig.MasterKey, "")
	as.Equal(config.SeseThreshold.AdultThreshold, 0.8)
	as.Equal(config.SeseThreshold.RacyThreshold, 0.6)
	as.Equal(config.LogLevel, int8(-1))

}

func TestJson(t *testing.T) {
	as := require.New(t)
	a := `[[[46.04093933105469,203.5409393310547],[197.71591186523438,203.625],[197.7681121826172,355.26812744140625],[46.784088134765625,355.875]],[[236.1346435546875,203.58839416503906],[387.7568664550781,203.6637725830078],[387.5313720703125,355.3664855957031],[236.1658477783203,355.91387939453125]]]`

	var b [][][]float64
	err := json.Unmarshal([]byte(a), &b)
	as.NoError(err)
}

func TestMap(t *testing.T) {
	m := make(map[string]*int)
	if m["a"] == nil {
		m["a"] = new(int)
	}
	fmt.Println(m["a"])
}
func TestPretty(t *testing.T) {
	pretty.Printf("%# v", GetConfig())
}
