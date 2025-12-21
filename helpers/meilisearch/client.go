package meilisearch

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

type Client struct {
	httpClient *http.Client
	Index      string
	BaseUrl    string
	MasterKey  string
	PrimaryKey string
}

func NewMeiliClient(baseUrl, index, masterKey string) *Client {
	return &Client{
		httpClient: &http.Client{},
		Index:      index,
		BaseUrl:    baseUrl,
		MasterKey:  masterKey,
	}
}
func validStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 299
}
func (c *Client) postJsonData(u string, data io.Reader, ignoreOutput bool) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, u, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.MasterKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.MasterKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out io.Writer
	outBuf := bytes.NewBuffer(nil)
	if ignoreOutput && validStatusCode(resp.StatusCode) {
		out = io.Discard
	} else {
		out = outBuf
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}
	if !validStatusCode(resp.StatusCode) {
		return nil, fmt.Errorf("POST %s http status code %d, error: %s", u, resp.StatusCode, outBuf.Bytes())
	}
	return outBuf.Bytes(), nil
}

func (c *Client) AddDocument(data any) error {
	buf := bytes.NewBuffer(nil)
	err := jsoniter.NewEncoder(buf).Encode(data)
	if err != nil {
		return err
	}
	u := fmt.Sprintf("%s/indexes/%s/documents?primaryKey=%s", c.BaseUrl, c.Index, c.PrimaryKey)
	_, err = c.postJsonData(u, buf, true)
	return err
}

type SearchQuery struct {
	Q      string   `json:"q"`
	Filter string   `json:"filter"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Sort   []string `json:"sort"`
}

func (c *Client) Search(q SearchQuery, result any) error {
	u := fmt.Sprintf("%s/indexes/%s/search", c.BaseUrl, c.Index)
	buf := bytes.NewBuffer(nil)
	err := jsoniter.NewEncoder(buf).Encode(q)
	if err != nil {
		return err
	}
	data, err := c.postJsonData(u, buf, false)
	if err != nil {
		return err
	}
	return jsoniter.NewDecoder(bytes.NewReader(data)).Decode(result)
}
