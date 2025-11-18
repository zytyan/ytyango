package ytdlp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
)

var Bin = "yt-dlp"
var reA = regexp.MustCompile(`(?i)/(BV[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+|av\d+)`)

type RunError struct {
	Err      error
	Stdout   string
	Stderr   string
	Args     []string
	ExitCode int
}

func (e *RunError) Error() string {
	argList := make([]string, 0, len(e.Args))
	for _, arg := range e.Args {
		argList = append(argList, strconv.Quote(arg))
	}
	args := strings.Join(argList, " ")
	return fmt.Sprintf("%s\nstderr=%s\nstdout=%s\nargs=%s", e.Err, e.Stderr, e.Stdout, args)
}

type Req struct {
	Url             string
	AudioOnly       bool
	Resolution      int
	EmbedMetadata   bool
	PostExec        string
	PriorityFormats []string
	WriteInfoJson   bool

	tmpPath  string
	prepared bool
}

type Resp struct {
	req         *Req
	FilePath    string
	InfoJson    map[string]interface{}
	InfoJsonErr error // 考虑到info json不是必须的，所以不返回错误，而是在这里附加上
}

func (c *Req) prepare() error {
	if c.prepared {
		return nil
	}
	c.prepared = true
	if c.Resolution == 0 {
		c.Resolution = 1080
	}
	var err error
	if c.tmpPath == "" {
		c.tmpPath, err = os.MkdirTemp("", "ytdlp")
		if err != nil {
			return err
		}
	}
	return nil

}

func (c *Req) args() []string {
	args := []string{c.Url,
		"-o",
		filepath.Join(c.tmpPath, "%(title).150B.%(ext)s"),
		"--quiet",
		"--print", "after_move:%(filepath)j",
		"--windows-filenames",
	}
	if strings.Contains(c.Url, "youtu") {
		// 目前就油管用到了这个能力，在B站启用反而会导致下载失败的问题
		args = append(args, "--format", `bestvideo+bestaudio/best`)
	}
	if c.WriteInfoJson {
		args = append(args, "--write-info-json")
	}
	// 分辨率
	formats := strings.Join(c.PriorityFormats, ":")
	args = append(args, "--format-sort", fmt.Sprintf(`+vcodec:%s,+acodec:flac:alac:wav:aiff:aac:mp4a:mp3,res:%d`, formats, c.Resolution))
	if c.AudioOnly {
		args = append(args, "--extract-audio", "--audio-format", "mp3", "--audio-quality", "0")
	}
	if c.EmbedMetadata {
		args = append(args, "--embed-metadata", "--embed-info-json", "--add-metadata")
	}
	if c.PostExec != "" {
		args = append(args, "--exec", c.PostExec)
	}

	return args
}

func (c *Req) Clean() error {
	if c == nil {
		return nil
	}
	return os.RemoveAll(c.tmpPath)
}

func getFirstFile(path string) (string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("读取目录 %s 失败", path)
	}
	for _, file := range files {
		if !file.IsDir() {
			firstFile := filepath.Join(path, file.Name())
			return firstFile, nil
		}
	}
	return "", fmt.Errorf("%s 目录中没有文件", path)
}

var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 10 * time.Second,
}

func getBilibiliVideoInfo(url string) (map[string]any, error) {
	location := url
	if strings.Contains(location, "b23.tv") {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		location = resp.Header.Get("Location")
		_ = resp.Body.Close()
		if location == "" {
			return nil, fmt.Errorf("cannot found %s", url)
		}
	}
	submatch := reA.FindStringSubmatch(location)
	if submatch == nil {
		return nil, fmt.Errorf("regex cannot found %s", location)
	}
	bv := submatch[1]
	reqUrl := ``
	switch bv[0] {
	case 'A', 'a':
		reqUrl = fmt.Sprintf(`https://api.bilibili.com/x/web-interface/view?aid=%s`, bv[2:])
	case 'B', 'b':
		reqUrl = fmt.Sprintf(`https://api.bilibili.com/x/web-interface/view?bvid=%s`, bv)
	default:
		return nil, fmt.Errorf("localtion AV/BV not in regular %s", location)
	}
	resp, err := client.Get(reqUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var m map[string]any
	err = jsoniter.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if code, ok := m["code"]; !ok || code != 0.0 {
		return nil, fmt.Errorf("code(%d) is not 0 %s", code, location)
	}
	tmp := m["data"].(map[string]interface{})
	result["title"] = tmp["title"]
	result["uploader"] = tmp["owner"]
	desc := tmp["desc"]
	descv2 := tmp["desc_v2"]
	if s, ok := desc.(string); ok && s != "" {
		result["description"] = s
	}
	if s, ok := descv2.([]any); ok {
		result["description"] = s[0].(map[string]any)["raw_text"]
	}
	return result, nil
}

func (c *Req) runWithCtxBBDown(ctx context.Context) (resp *Resp, err error) {
	resp = &Resp{req: c}
	wg := sync.WaitGroup{}
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				resp.InfoJsonErr = fmt.Errorf("get BilibiliVideoInfo error: %s", r)
			}
		}()
		resp.InfoJson, resp.InfoJsonErr = getBilibiliVideoInfo(c.Url)
		if resp.InfoJsonErr != nil {
			log.Println(resp.InfoJsonErr)
		}
	})

	tmp, err := os.MkdirTemp("", "ytdlp-*")
	if err != nil {
		return nil, err
	}
	tmp, err = filepath.Abs(tmp)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "/usr/local/bin/dotnet-tools/BBDown", c.Url, "-e", "hevc,av1,avc", "-q", "720P 高清", "-F", fmt.Sprintf("%s/<bvid>", tmp))
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err = cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && errors.As(err, &exitErr) {
		return resp, &RunError{
			Err:      err,
			Stdout:   outBuf.String(),
			Stderr:   errBuf.String(),
			ExitCode: exitErr.ExitCode(),
			Args:     cmd.Args,
		}
	}
	wg.Wait()
	resp.FilePath, err = getFirstFile(tmp)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func (c *Req) runWithCtx(ctx context.Context) (resp *Resp, err error) {
	if strings.Contains(c.Url, "b23.tv") || strings.Contains(c.Url, "bilibili.com") {
		return c.runWithCtxBBDown(ctx)
	}
	resp = &Resp{req: c}
	err = c.prepare()
	if err != nil {
		return resp, err
	}
	args := c.args()
	cmd := exec.CommandContext(ctx, Bin, args...)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err = cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && errors.As(err, &exitErr) {
		return resp, &RunError{
			Err:      err,
			Stdout:   outBuf.String(),
			Stderr:   errBuf.String(),
			ExitCode: exitErr.ExitCode(),
			Args:     args,
		}
	}
	err = jsoniter.Unmarshal(outBuf.Bytes(), &resp.FilePath)
	if err != nil {
		return resp, err
	}
	if c.WriteInfoJson {
		ext := filepath.Ext(resp.FilePath)
		infoJsonPath := resp.FilePath[:len(resp.FilePath)-len(ext)] + ".info.json"
		infoJsonFile, err := os.ReadFile(infoJsonPath)
		if err != nil {
			resp.InfoJsonErr = err
		} else {
			err = jsoniter.Unmarshal(infoJsonFile, &resp.InfoJson)
			if err != nil {
				resp.InfoJsonErr = err
			}
		}
	}
	return resp, nil
}

func (c *Req) RunWithTimeout(timeout time.Duration) (*Resp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.runWithCtx(ctx)
}

func (r *Resp) Uploader() string {
	if !r.req.WriteInfoJson || r.InfoJson == nil {
		return ""
	}
	uploader, _ := r.InfoJson["uploader"].(string)
	return uploader
}

func (r *Resp) Title() string {
	if !r.req.WriteInfoJson || r.InfoJson == nil {
		return ""
	}
	title, _ := r.InfoJson["title"].(string)
	return title
}

func (r *Resp) Description() string {
	if !r.req.WriteInfoJson || r.InfoJson == nil {
		return ""
	}
	description, _ := r.InfoJson["description"].(string)
	return description
}

func (r *Resp) Thumbnail() (string, error) {
	if !r.req.WriteInfoJson || r.InfoJson == nil {
		return "", nil
	}
	thumbnail, _ := r.InfoJson["thumbnail"].(string)
	if thumbnail == "" {
		return "", nil
	}
	return ExtractFirstFrame(thumbnail)
}

func ExtractFirstFrame(path string) (string, error) {
	outPath := path + ".jpg"
	cmd := exec.Command("ffmpeg", "-i", path, "-vframes", "1", "-q:v", "2", outPath)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && errors.As(err, &exitErr) {
		return "", &RunError{
			Err:      err,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: exitErr.ExitCode(),
		}
	} else if err != nil {
		return "", err
	}
	return outPath, nil
}
