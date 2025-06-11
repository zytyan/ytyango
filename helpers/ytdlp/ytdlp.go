package ytdlp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var Bin = "yt-dlp"

type RunError struct {
	Err      error
	Stdout   string
	Stderr   string
	ExitCode int
}

func (e *RunError) Error() string {
	return fmt.Sprintf("%s\nstderr=%s\nstdout=%s", e.Err, e.Stderr, e.Stdout)
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
		//"--format", `bestvideo[ext=mp4]+bestaudio[ext=m4a]/best`,
		"--windows-filenames",
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

func (c *Req) runWithCtx(ctx context.Context) (resp *Resp, err error) {
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
