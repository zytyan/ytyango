package backend

import (
	"bytes"
	"io"
	"log"
	api "main/http/backend/ogen"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testServer *api.Server

const baseAddr = "127.0.0.1:9892"

var baseUrl = "http://" + baseAddr + "/"

func TestMain(m *testing.M) {
	var err error
	testServer, err = NewServer(nil)
	if err != nil {
		panic(err)
	}
	ln, err := net.Listen("tcp", baseAddr)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err := http.Serve(ln, testServer)
		if err != nil {
			log.Fatal(err)
		}
	}()
	time.Sleep(time.Millisecond * 10)
	code := m.Run()
	os.Exit(code)
}

var hAuthData = "query_id=AAGhdeMLAAAAAKF14wu7BOmF&user=%7B%22" +
	"id%22%3A199456161%2C%22first_name%22%3A%22z%22%2C" +
	"%22last_name%22%3A%22%22%2C%22username%22%3A%22Yt" +
	"yan%22%2C%22language_code%22%3A%22zh-hans%22%2C%2" +
	"2is_premium%22%3Atrue%2C%22allows_write_to_pm%22%3" +
	"Atrue%7D&auth_date=1695737566&hash=0cd9a1d70d6b6c" +
	"d83630e455eb72d89dfd955a5068272c621402684358f32d68"

func TestSecurity(t *testing.T) {
	as := assert.New(t)
	data := []byte(`{"user_ids": [0]}`)
	path := baseUrl + "users/info"
	resp, err := http.Post(path, "application/json", bytes.NewReader(data))
	as.NoError(err)
	as.Equal(401, resp.StatusCode)
	respData, err := io.ReadAll(resp.Body)
	as.NoError(err)
	as.Equal(`{"error_message":"operation GetUsersInfo: security \"\": security requirement is not satisfied"}`, string(respData))
	req, err := http.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	as.NoError(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Telegram-Init-Data", hAuthData)
	resp, err = http.DefaultClient.Do(req)
	as.NoError(err)
	as.Equal(200, resp.StatusCode)
	respData, err = io.ReadAll(resp.Body)
	as.NoError(err)
	as.Equal(`{"users":[{"id":0,"name":"","error":"user id invalid"}]}`, string(respData))
	path = baseUrl + "users/1000/avatar"
	resp, err = http.Get(path + "?tgauth=" + url.QueryEscape("hash=00000000"))
	as.NoError(err)
	as.Equal(401, resp.StatusCode)
	respData, err = io.ReadAll(resp.Body)

	path = baseUrl + "users/1000/avatar"
	resp, err = http.Get(path + "?tgauth=" + url.QueryEscape(hAuthData))
	as.NoError(err)
	respData, err = io.ReadAll(resp.Body)
	as.Equal(404, resp.StatusCode)
	as.Equal(`{"message":"user has no profile photo"}`, string(respData))
}
