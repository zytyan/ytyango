package azure

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestSese(t *testing.T) {
	key := os.Getenv("AZURE_MODERATOR_KEY")
	endpoint := os.Getenv("AZURE_MODERATOR_ENDPOINT")
	as := require.New(t)
	client1 := NewClient(endpoint, key, ContentModeratorPath)
	client := Moderator{Client: *client1}
	res, err := client.EvalFile(`test/racy.png`)
	as.NoError(err)
	as.NotNil(res)
	as.True(res.IsImageRacyClassified)
}

func TestOcr(t *testing.T) {
	key := os.Getenv("AZURE_OCR_KEY")
	endpoint := os.Getenv("AZURE_OCR_ENDPOINT")
	as := require.New(t)
	client1 := NewClient(endpoint, key, OcrPath)
	client := Ocr{Client: *client1,
		//Language: "zh-Hans",
		Features: "Read",
		ApiVer:   "2023-02-01-preview",
	}
	res, err := client.OcrFile(`test/ocr_test.png`)
	as.NoError(err)
	as.NotNil(res)
	as.Contains(res.ReadResult.Content, "函数应用")
}
