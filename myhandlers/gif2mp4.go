package myhandlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Semaphore struct {
	sem chan struct{}
}

func (s *Semaphore) AcquireWithTimeout(timeout time.Duration) bool {
	select {
	case s.sem <- struct{}{}:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (s *Semaphore) Acquire() {
	s.sem <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.sem
}

func NewSemaphore(n int) *Semaphore {
	return &Semaphore{sem: make(chan struct{}, n)}
}

var sema = NewSemaphore(1)

func gif2mp4(gifFile, mp4File string) error {
	// mp4 要求长宽必须是偶数，所以使用除2再乘2的方式来保证长宽是偶数
	// command on shell: ffmpeg -i test.gif -movflags faststart -pix_fmt yuv420p -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2" test.mp4
	cmd := exec.Command("ffmpeg",
		"-i", gifFile,
		"-movflags", "faststart",
		"-pix_fmt", "yuv420p",
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
		mp4File)
	err := cmd.Run()
	if err != nil {
		log.Warnf("convert gif to mp4 failed: %s", err)
		return err
	}
	return nil
}

func downloadSinaGif(sinaUrl string) (gifFile string, err error) {
	get, err := http.Get(sinaUrl)
	if err != nil {
		return "", err
	}
	urlObj, err := url.Parse(sinaUrl)
	if err != nil {
		return "", err
	}
	// 获取url中的文件名
	pathBuf := strings.SplitN(urlObj.Path, "/", -1)
	gifFile = fmt.Sprintf("/tmp/%s", pathBuf[len(pathBuf)-1])
	defer get.Body.Close()
	if get.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code not ok")
	}
	file, err := os.Create(gifFile)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = io.Copy(file, get.Body)
	if err != nil {
		return "", err
	}
	return gifFile, nil
}

var sinaGifRe = regexp.MustCompile(`https?://wx\d+\.sinaimg\.cn/[\w/\-_]+\.gif`)

func HasSinaGif(msg *gotgbot.Message) bool {
	return sinaGifRe.MatchString(getTextMsg(msg))
}

func gif2Mp4(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	text := getText(ctx)
	if text == "" {
		return
	}
	gifUrl := sinaGifRe.FindString(text)
	if gifUrl == "" {
		return
	}
	gifFile, err := downloadSinaGif(gifUrl)
	if err != nil {
		return err
	}
	mp4File := gifFile + ".mp4"
	err = gif2mp4(gifFile, mp4File)
	if err != nil {
		return err
	}
	_, err = bot.SendVideo(ctx.EffectiveChat.Id, gotgbot.InputFile(mp4File), nil)
	if err != nil {
		return err
	}
	return
}

func Gif2Mp4(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	go func() {
		if !sema.AcquireWithTimeout(5 * time.Second) {
			log.Warnf("gif2mp4 semaphore timeout")
			_, err := ctx.EffectiveMessage.Reply(bot, "转换失败，请稍后再试", nil)
			if err != nil {
				log.Warnf("reply message failed, err: %s", err)
				return
			}
			return
		}
		defer sema.Release()
		err = gif2Mp4(bot, ctx)
	}()
	return
}