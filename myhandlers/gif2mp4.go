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
	"path/filepath"
	"regexp"
	"strings"
)

func gif2mp4(gifFile, mp4File string) error {
	// mp4 要求长宽必须是偶数，所以使用除2再乘2的方式来保证长宽是偶数
	// command on shell: ffmpeg -i test.gif -movflags faststart -pix_fmt yuv420p -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2" test.mp4
	cmd := exec.Command("ffmpeg",
		"-y",
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
	log.Infof("download gif from %s", sinaUrl)
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
	gifFile = filepath.Join(os.TempDir(), pathBuf[len(pathBuf)-1])
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
	gifFile, err = filepath.Abs(gifFile)
	if err != nil {
		return "", err
	}
	log.Infof("download gif to %s", gifFile)
	return gifFile, nil
}

var sinaGifRe = regexp.MustCompile(`https?://wx\d+\.(sinaimg\.cn|moyu\.im)/[\w/\-_]+\.gif`)

func HasSinaGif(msg *gotgbot.Message) bool {
	if g, err := getGroupInfo(msg.Chat.Id); err != nil || !g.AutoCvtBili {
		return false
	}
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
	gifUrl = strings.ReplaceAll(gifUrl, "moyu.im", "sinaimg.cn")
	gifFile, err := downloadSinaGif(gifUrl)
	if err != nil {
		return err
	}
	mp4File := gifFile + ".mp4"
	err = gif2mp4(gifFile, mp4File)
	if err != nil {
		return err
	}
	defer os.Remove(mp4File)
	defer os.Remove(gifFile)
	_, err = bot.SendVideo(ctx.EffectiveChat.Id, fileSchema(mp4File), nil)
	if err != nil {
		return err
	}
	return
}

func Gif2Mp4(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	go func() {
		err = gif2Mp4(bot, ctx)
		if err != nil {
			log.Warnf("gif2mp4 failed: %s", err)
		}
	}()
	return
}
