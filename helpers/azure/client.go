package azure

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

//goland:noinspection GoUnusedConst
const (
	ContentModeratorPath   = "/contentmoderator/moderate/v1.0/ProcessImage/Evaluate"
	ContentModeratorV2Path = "/contentsafety/image:analyze?api-version=2024-09-01"
	OcrPath                = "/computervision/imageanalysis:analyze"
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

func (c *Client) reqWithAuth(method, contentType string) *http.Request {
	urlPath := fmt.Sprintf("%s%s", c.endpoint, c.path)
	request, err := http.NewRequest(method, urlPath, nil)
	if err != nil {
		panic(err) // 理论上不会有问题
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Add("Ocp-Apim-Subscription-Key", c.apiKey)
	return request
}

func unmarshalResponse(resp *http.Response, v any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP Response error(%d): %s", resp.StatusCode, body)
	}
	err = jsoniter.Unmarshal(body, v)
	if err != nil {
		return err
	}
	return nil
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
	req := m.reqWithAuth(http.MethodPost, "image/jpeg")
	req.Body = file
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	res := &ModeratorResult{}
	err = unmarshalResponse(resp, res)
	return res, err
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
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return o.ocr(file, stat.Size())
}

func (o *Ocr) OcrData(data []byte) (*OcrResult, error) {
	return o.ocr(io.NopCloser(bytes.NewReader(data)), int64(len(data)))
}

func (o *Ocr) ocr(body io.ReadCloser, contentLength int64) (*OcrResult, error) {
	req := o.reqWithAuth(http.MethodPost, "image/jpeg")
	req.Body = body
	req.ContentLength = contentLength
	q := req.URL.Query()
	q.Add("api-version", o.ApiVer)
	if o.Features != "" {
		q.Add("features", o.Features)
	}
	if o.Language != "" {
		q.Add("language", o.Language)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	res := &OcrResult{}
	err = unmarshalResponse(resp, res)
	return res, err
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

//goland:noinspection GoUnusedConst
const (
	ModerateV2CatHate     = "Hate"
	ModerateV2CatSelfHarm = "SelfHarm"
	ModerateV2CatViolence = "Violence"
	ModerateV2CatSexual   = "Sexual"
)

type ModeratorV2Result struct {
	CategoriesAnalysis []struct {
		Category string `json:"category"`
		Severity int    `json:"severity"`
	} `json:"categoriesAnalysis"`
}

type moderatorV2Param struct {
	Image struct {
		Content string `json:"content"`
	} `json:"image"`
	Categories []string `json:"categories"`
	OutputType string   `json:"outputType"`
}

type ModeratorV2 struct {
	Client
	Categories []string `json:"categories"`
	OutputType string   `json:"outputType"`
}

func (m *ModeratorV2) EvalFile(path string) (*ModeratorV2Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return m.EvalData(data)
}

func (m *ModeratorV2) EvalData(data []byte) (*ModeratorV2Result, error) {
	req := m.reqWithAuth(http.MethodPost, "application/json")
	b64Data := base64.StdEncoding.EncodeToString(data)
	param := moderatorV2Param{
		Categories: m.Categories,
		OutputType: m.OutputType,
	}
	param.Image.Content = b64Data
	body, err := jsoniter.Marshal(&param)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	result := &ModeratorV2Result{}
	err = unmarshalResponse(resp, result)
	return result, err
}

func (r *ModeratorV2Result) GetSeverityByCategory(category string) int {
	for _, analysis := range r.CategoriesAnalysis {
		if analysis.Category == category {
			return analysis.Severity
		}
	}
	return -1
}
