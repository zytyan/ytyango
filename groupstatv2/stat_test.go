package groupstatv2

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	as := assert.New(t)
	days := newGroupStatTwoDays()
	days.Today.MessageCount.Inc()
	days.Today.MessageCount.Inc()
	days.Today.MsgCountByTime[10].Add(4321)

	userMsgStat := &UserMsgStat{}
	days.Today.UserMsgStat.Store(123, userMsgStat)
	userMsgStat.MsgCount.Add(100)
	userMsgStat.MsgLen.Add(200)

	// 测试是否能正确序列化
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(days)
	as.NoError(err)
	data := buf.Bytes()

	// 测试能否正确反序列化
	daysNew := &GroupStatTwoDays{}
	err = gob.NewDecoder(bytes.NewReader(data)).Decode(daysNew)
	as.NoError(err)

	// 测试反序列化后是否能正确提取值
	as.Equal(days.Today.MessageCount.Load(), daysNew.Today.MessageCount.Load())
	as.Equal(days.Today.MsgCountByTime[10].Load(), daysNew.Today.MsgCountByTime[10].Load())
	ums2, loaded := daysNew.Today.UserMsgStat.Load(123)
	as.True(loaded)
	as.Equal(int64(100), ums2.MsgCount.Load())
	as.Equal(int64(200), ums2.MsgLen.Load())
	as.NotEqualf(days.Today, daysNew.Today, "days与daysNew实际上是同一个变量，无法正确测试！")

	// 测试反序列化后再序列化，其值是否相等
	buf2 := &bytes.Buffer{}
	err = gob.NewEncoder(buf2).Encode(daysNew)
	as.NoError(err)
	as.Equal(data, buf2.Bytes())
}

func TestGetGroup(t *testing.T) {
	as := assert.New(t)
	days := newGroupStatTwoDays()
	days.Today.MessageCount.Inc()
	days.Today.MsgCountByTime[10].Add(4321)
	days.mu.Lock()
	days.mu.Unlock()
	as.Nil(days.Yesterday)
}

func TestGlobalEncodeDecode(t *testing.T) {
	as := assert.New(t)
	_ = GetGroup(1)
	_ = GetGroup(2)
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(&groupStat)
	as.NoError(err)
	err = gob.NewDecoder(buf).Decode(&groupStat)
	as.NoError(err)
	_, loaded := groupStat.Load(1)
	as.True(loaded)
	_, loaded = groupStat.Load(3)
	as.False(loaded)
}
