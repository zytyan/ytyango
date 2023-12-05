package ytdlp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	return fmt.Sprintf("%s\n%s", e.Err, e.Stderr)
}

type Config struct {
	Url             string
	AudioOnly       bool
	Resolution      int
	EmbedMetadata   bool
	PostExec        string
	PriorityFormats []string

	tmpPath         string
	preparedDefault bool
}

func (c *Config) PrepareDefault() error {
	if c.preparedDefault {
		return nil
	}
	c.preparedDefault = true
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
func (c *Config) args() ([]string, error) {
	err := c.PrepareDefault()
	if err != nil {
		return nil, err
	}
	args := []string{c.Url,
		"-o",
		filepath.Join(c.tmpPath, "%(title).150B.%(ext)s"),
		"--format", `bestvideo*+bestaudio/best`,
		"--windows-filenames",
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

	return args, nil
}

func (c *Config) Clean() error {
	if c == nil {
		return nil
	}
	return os.RemoveAll(c.tmpPath)
}

func (c *Config) RunWithCtx(ctx context.Context) error {
	args, err := c.args()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, Bin, args...)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err = cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && errors.As(err, &exitErr) {
		return &RunError{
			Err:      err,
			Stdout:   outBuf.String(),
			Stderr:   errBuf.String(),
			ExitCode: exitErr.ExitCode(),
		}
	}
	return err
}

func (c *Config) RunWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.RunWithCtx(ctx)
}

func (c *Config) Run() error {
	return c.RunWithCtx(context.Background())
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
