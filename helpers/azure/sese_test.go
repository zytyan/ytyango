package azure

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSese(t *testing.T) {
	client1 := NewClient("https://sese-detect.cognitiveservices.azure.com", "***REMOVED***", ContentModeratorPath)
	client := Moderator{Client: *client1}
	filepath, err := client.EvalFile(`D:\PhoneSync\BittorrentDownload\01-10\6-习呆呆\图片\初音\www.hauntedfartfart.tumblr.com (26).jpg`)
	fmt.Println(filepath, err)
	if err != nil {
		t.Fail()
		return
	}

}

func TestOcr(t *testing.T) {
	as := require.New(t)
	client1 := NewClient("https://bot-cv.cognitiveservices.azure.com", "***REMOVED***", OcrPath)
	client := Ocr{Client: *client1,
		//Language: "zh-Hans",
		Features: "Read",
		ApiVer:   "2023-02-01-preview",
	}
	res, err := client.OcrFile(`D:\PhoneSync\ehviewer\1417339-[りとるほっぱー+Ziggurat (橋広こう)] さえちゃんの初体験3～勝手に悶絶睡眠姦～ [中国翻訳] [DL版]\00000003.jpg`)
	as.NoError(err)
	as.NotEmptyf(res, "%+v", res)
}
