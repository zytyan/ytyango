package bili

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExtractLink(t *testing.T) {
	as := require.New(t)
	text := `https://b23.tv/rg1TQ3N`
	indexList := keyword.FindAllStringIndex(text, -1)
	fmt.Println(indexList)
	as.Len(indexList, 1)
}
func TestPure(t *testing.T) {
	as := require.New(t)
	text := `https://b23.tv/rg1TQ3N`
	prepare, err := ContainsBiliLinkAndTryPrepare(text)
	fmt.Println(prepare, err)
	as.NoError(err)
	as.True(prepare.NeedConvert())
	as.Len(prepare, 1)
}

func TestOneWithComment(t *testing.T) {
	as := require.New(t)
	text := `this is a comment https://b23.tv/rg1TQ3N`
	prepare, err := ContainsBiliLinkAndTryPrepare(text)
	fmt.Println(prepare, err)
	as.NoError(err)
	as.True(prepare.NeedConvert())
	as.Len(prepare, 2)
}

func TestHttp(t *testing.T) {
	as := require.New(t)
	text := `http://b23.tv/rg1TQ3N`
	prepare, err := ContainsBiliLinkAndTryPrepare(text)
	fmt.Println(prepare, err)
	as.NoError(err)
	as.True(prepare.NeedConvert())
	as.Len(prepare, 1)
}
