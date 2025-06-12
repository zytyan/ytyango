package imgproc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestName(t *testing.T) {
	if !webpEnabled {
		t.Skip("webp not enabled")
		return
	}
	as := assert.New(t)
	err := encodeToWebp(prprBase("photo_2021-06-01_02-16-26.jpg"), "prpr.webp")
	as.Nil(err)
}
