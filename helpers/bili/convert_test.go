package bili

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

var testB23Link = `https://b23.tv/azH0KMi`
var testBili2233Link = `https://bili2233.cn/azH0KMi`
var testMallB23Link = `https://b23.tv/S62FYLs`

func TestAvToBv(t *testing.T) {
	as := require.New(t)
	as.Equal("/BV1xx411c7mD", av2bv("/av2"))
}

func TestBvToAv(t *testing.T) {
	as := require.New(t)
	as.Equal("/av2", bv2av("/BV1xx411c7mD"))
}

func TestExtractHttpLink(t *testing.T) {
	as := require.New(t)

	c, err := ConvertBilibiliLinks(testB23Link)
	as.NoError(err)
	as.True(c.CanConvert())
}

func TestPure(t *testing.T) {
	as := require.New(t)

	prepare, err := ConvertBilibiliLinks(testB23Link)
	as.NoError(err)
	as.True(prepare.CanConvert())
	as.True(prepare.HasBv)
	as.True(prepare.HasAv)
	as.Equal(`https://www.bilibili.com/video/BV166Fke1E5m?p=1`, prepare.BvText)
	prepare, err = ConvertBilibiliLinks(testBili2233Link)
	as.NoError(err)
	as.True(prepare.CanConvert())
	as.True(prepare.HasBv)
	as.True(prepare.HasAv)
	as.Equal(`https://www.bilibili.com/video/av113933939642269?p=1`, prepare.AvText)

}

func TestMall(t *testing.T) {
	as := require.New(t)
	prepare, err := ConvertBilibiliLinks(testMallB23Link)
	as.NoError(err)
	fmt.Println(prepare)
	as.True(prepare.CanConvert())

	as.True(prepare.HasBv)
	as.False(prepare.HasAv)
	as.Equal("https://mall.bilibili.com/detail.html?itemsId=10664158", prepare.BvText)
}

func TestOneWithComment(t *testing.T) {
	as := require.New(t)
	text := `this is a comment ` + testB23Link
	links, err := ConvertBilibiliLinks(text)
	as.NoError(err)
	as.True(links.CanConvert())
	as.Equal(`this is a comment https://www.bilibili.com/video/BV166Fke1E5m?p=1`, links.BvText)
}

var bilibiliLink = `https://www.bilibili.com/video/av113933939642269/?buvid=A8B976&is_story_h5=false&p=1&`

func TestBilibiliLink(t *testing.T) {
	as := require.New(t)
	links, err := ConvertBilibiliLinks(bilibiliLink)
	as.NoError(err)
	as.True(links.CanConvert())
	as.True(links.HasBv)
	as.True(links.HasAv)
	as.Equal(`https://www.bilibili.com/video/BV166Fke1E5m/?p=1`, links.BvText)
	as.Equal(`https://www.bilibili.com/video/av113933939642269/?p=1`, links.AvText)
}
func TestNoNeedConvert(t *testing.T) {
	as := require.New(t)
	links, err := ConvertBilibiliLinks("https://www.bilibili.com/video/BV166Fke1E5m/?p=1")
	as.NoError(err)
	as.False(links.NeedClean)
}
