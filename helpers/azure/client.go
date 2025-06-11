package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	ContentModeratorPath = "/contentmoderator/moderate/v1.0/ProcessImage/Evaluate"
	OcrPath              = "/computervision/imageanalysis:analyze"
)

type ResponseError struct {
	Error struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
}
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
	ResponseError
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
	ResponseError

	ModelVersion string `json:"modelVersion,omitempty"`
	Metadata     struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
	} `json:"metadata,omitempty"`
	ReadResult struct {
		Blocks []struct {
			Lines []struct {
				Text            string `json:"text,omitempty"`
				BoundingPolygon []struct {
					X int `json:"x,omitempty"`
					Y int `json:"y,omitempty"`
				} `json:"boundingPolygon,omitempty"`
				Words []struct {
					Text            string `json:"text,omitempty"`
					BoundingPolygon []struct {
						X int `json:"x,omitempty"`
						Y int `json:"y,omitempty"`
					} `json:"boundingPolygon,omitempty"`
					Confidence float64 `json:"confidence,omitempty"`
				} `json:"words,omitempty"`
			} `json:"lines,omitempty"`
		} `json:"blocks,omitempty"`
	} `json:"readResult,omitempty"`
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
type ImageAnalysisResult struct {
	ResponseError
	CategoriesAnalysis []struct {
		Category string `json:"category"`
		Severity int    `json:"severity"`
	} `json:"categoriesAnalysis"`
}

func (r *ResponseError) HasError() bool {
	return r.Error.Code == "" || r.Error.Code == "0"
}
func (r *ResponseError) ToError() error {
	if r.HasError() {
		return nil
	}
	return fmt.Errorf("azure error, code = %s, msg = %s", r.Error.Code, r.Error.Message)
}
func (r *OcrResult) Text() string {
	buf := strings.Builder{}
	for _, block := range r.ReadResult.Blocks {
		for _, line := range block.Lines {
			buf.WriteString(line.Text)
			buf.WriteByte('\n')
		}
		buf.WriteString("\n\n")
	}
	return buf.String()

}
