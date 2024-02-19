package bili

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExtractLink(t *testing.T) {
	as := require.New(t)
	text := `https://b23.tv/rg1TQ3N`
	indexList := keyword.FindAllStringIndex(text, -1)
	as.Len(indexList, 1)
}

func TestExtractHttpLink(t *testing.T) {
	as := require.New(t)

	text := `http://b23.tv/rg1TQ3N`
	parsed, err := ContainsBiliLinkAndTryPrepare(text)
	as.NoError(err)
	as.True(parsed.NeedConvert())
	bv, err := parsed.ToBv()
	as.NoError(err)
	as.Equal(`https://www.bilibili.com/video/BV1yp4y1M7Mi?p=1`, bv)
}

func TestPure(t *testing.T) {
	as := require.New(t)
	text := `https://b23.tv/rg1TQ3N`
	prepare, err := ContainsBiliLinkAndTryPrepare(text)
	as.NoError(err)
	as.True(prepare.NeedConvert())
	as.Len(prepare, 1)
}
func TestMall(t *testing.T) {
	as := require.New(t)
	text := `https://b23.tv/58xhm9y`
	prepare, err := ContainsBiliLinkAndTryPrepare(text)
	as.NoError(err)
	as.True(prepare.NeedConvert())
	as.Len(prepare, 1)
	link, err := prepare.ToBv()
	as.NoError(err)
	as.Equal("https://mall.bilibili.com/detail.html?itemsId=10247954", link)
}
func TestOneWithComment(t *testing.T) {
	as := require.New(t)
	text := `this is a comment https://b23.tv/rg1TQ3N`
	prepare, err := ContainsBiliLinkAndTryPrepare(text)
	as.NoError(err)
	as.True(prepare.NeedConvert())
	as.Len(prepare, 2)
	as.Equal(NotLink, prepare[0].LinkType)
	as.Equal(B23, prepare[1].LinkType)
}

func TestHttp(t *testing.T) {
	as := require.New(t)
	text := `http://b23.tv/rg1TQ3N`
	prepare, err := ContainsBiliLinkAndTryPrepare(text)
	as.NoError(err)
	as.True(prepare.NeedConvert())
	as.Len(prepare, 1)
}

func TestLongList(t *testing.T) {
	as := require.New(t)
	text := `https://www.bilibili.com/video/BV1Zi4y1H7JR/?spm_id_from=333.`
	parsed, err := ContainsBiliLinkAndTryPrepare(text)
	as.NoError(err)
	as.True(parsed.NeedConvert())
	bv, err := parsed.ToBv()
	as.NoError(err)
	as.Equal("https://www.bilibili.com/video/BV1Zi4y1H7JR/", bv)
}

func TestBilibiliCleanParams(t *testing.T) {
	as := require.New(t)
	text := `https://www.bilibili.com/video/BV1Zi4y1H7JR/?spm_id_from=333.`
	clean, err := BilibiliCleanParams(text)
	as.NoError(err)
	as.Equal("https://www.bilibili.com/video/BV1Zi4y1H7JR/", clean)
}
func TestKeyword_FindAllStringIndex(t *testing.T) {
	as := require.New(t)
	text := `https://www.bilibili.com/video/BV1Zi4y1H7JR/?spm_id_from=333.`
	indexList := keyword.FindAllStringIndex(text, -1)
	as.Len(indexList, 1)
	as.Equal(0, indexList[0][0])
	as.Equal(len(text), indexList[0][1])
}
