package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	ContentModeratorPath = "/contentmoderator/moderate/v1.0/ProcessImage/Evaluate"
	OcrPath              = "/computervision/imageanalysis:analyze"
)

type Client struct {
	client   http.Client
	endpoint string
	apiKey   string
	path     string
}

func NewClient(endpoint string, apiKey string, path string) *Client {
	return &Client{apiKey: apiKey, endpoint: endpoint, path: path}
}

func (c *Client) reqWithAuth() *http.Request {
	urlPath := fmt.Sprintf("%s%s", c.endpoint, c.path)
	request, err := http.NewRequest(http.MethodGet, urlPath, nil)
	if err != nil {
		return nil
	}
	request.Header.Add("Ocp-Apim-Subscription-Key", c.apiKey)
	return request
}

type ModeratorResult struct {
	AdultClassificationScore float64 `json:"AdultClassificationScore"`
	IsImageAdultClassified   bool    `json:"IsImageAdultClassified"`
	RacyClassificationScore  float64 `json:"RacyClassificationScore"`
	IsImageRacyClassified    bool    `json:"IsImageRacyClassified"`
	Result                   bool    `json:"Result"`
	AdvancedInfo             []any   `json:"AdvancedInfo"`
	Status                   struct {
		Code        int    `json:"Code"`
		Description string `json:"Description"`
		Exception   any    `json:"Exception"`
	} `json:"Status"`
	TrackingID string `json:"TrackingId"`
}

type Moderator struct {
	Client
}

func (m *Moderator) EvalFile(path string) (*ModeratorResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	req := m.reqWithAuth()
	req.Method = http.MethodPost
	req.Header.Add("Content-Type", "image/jpeg")
	req.Body = file
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d, error body: %s", resp.StatusCode, data)
	}
	res := ModeratorResult{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

type Ocr struct {
	Client
	ApiVer   string
	Language string
	Features string
}

type OcrResult struct {
	ReadResult struct {
		StringIndexType string `json:"stringIndexType"`
		Content         string `json:"content"`
		Pages           []struct {
			Height     float64 `json:"height"`
			Width      float64 `json:"width"`
			Angle      float64 `json:"angle"`
			PageNumber int     `json:"pageNumber"`
			Words      []struct {
				Content     string    `json:"content"`
				BoundingBox []float64 `json:"boundingBox"`
				Confidence  float64   `json:"confidence"`
				Span        struct {
					Offset int `json:"offset"`
					Length int `json:"length"`
				} `json:"span"`
			} `json:"words"`
			Spans []struct {
				Offset int `json:"offset"`
				Length int `json:"length"`
			} `json:"spans"`
			Lines []struct {
				Content     string    `json:"content"`
				BoundingBox []float64 `json:"boundingBox"`
				Spans       []struct {
					Offset int `json:"offset"`
					Length int `json:"length"`
				} `json:"spans"`
			} `json:"lines"`
		} `json:"pages"`
		Styles []struct {
			IsHandwritten bool `json:"isHandwritten"`
			Spans         []struct {
				Offset int `json:"offset"`
				Length int `json:"length"`
			} `json:"spans"`
			Confidence float64 `json:"confidence"`
		} `json:"styles"`
		ModelVersion string `json:"modelVersion"`
	} `json:"readResult"`
	ModelVersion string `json:"modelVersion"`
	Metadata     struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"metadata"`
}

func (o *Ocr) OcrFile(path string) (*OcrResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	req := o.reqWithAuth()
	req.Header.Set("Content-Type", "image/jpeg")
	req.Method = http.MethodPost
	//req.TransferEncoding = []string{"identity"}
	req.Body = file
	q := req.URL.Query()
	q.Add("api-version", o.ApiVer)
	if o.Features != "" {
		q.Add("features", o.Features)
	}
	if o.Language != "" {
		q.Add("language", o.Language)

	}
	req.URL.RawQuery = q.Encode()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	req.ContentLength = stat.Size()
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d, error body: %s", resp.StatusCode, data)
	}
	res := OcrResult{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

type Error struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
