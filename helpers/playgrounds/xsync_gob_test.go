package playgrounds

import (
	"bytes"
	"encoding/gob"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func gobEncode(m interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(m)
	return buf.Bytes(), err
}
func TestXsyncGob(t *testing.T) {
	as := assert.New(t)
	m := xsync.NewMapOf[string, string]()
	m.Store("a", "b")
	m.Store("c", "d")
	m.Store("e", "f")
	data, err := gobEncode(m)
	// xsync.MapOf 根本不能编码，也没有适配编码功能
	as.NotNil(err)
	as.NotNil(data)
}

func TestJsonInt(t *testing.T) {
	as := assert.New(t)
	m := make(map[int64]int)
	m[1] = 2
	m[3] = 4
	data, err := jsoniter.Marshal(m)
	as.Nil(err)
	as.NotNil(data)
	m2 := make(map[int64]int)
	fmt.Println(string(data))
	err = jsoniter.Unmarshal(data, &m2)
	as.Nil(err)
	as.NotNil(m2)
	as.Equal(m, m2)
}
