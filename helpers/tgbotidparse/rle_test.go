package tgbotidparse

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestRleEncodeAndDecode(t *testing.T) {

	as := require.New(t)

	// 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	data1 := []byte{0x00, 0x00, 0x00, 0x07}
	encoded := rleEncode(data1)
	as.Equal(rleDecode(encoded), data1)

}

func TestTlStringShort(t *testing.T) {
	as := require.New(t)
	url1 := []byte("https://localhost:8888")
	packed := packTLString(url1)
	unpacked, err, _ := unpackTLString(packed)
	as.Nil(err)
	as.Equal(url1, unpacked)
}
func TestTlStringLong(t *testing.T) {
	as := require.New(t)
	url1 := []byte(strings.Repeat("https://localhost:8888", 100))
	packed := packTLString(url1)
	unpacked, err, _ := unpackTLString(packed)
	as.Nil(err)
	as.Equal(url1, unpacked)
}
