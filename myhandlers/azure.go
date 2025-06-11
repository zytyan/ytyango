package myhandlers

import (
	"errors"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"golang.org/x/time/rate"
	"image"
	"main/globalcfg"
	"main/helpers/azure"
	"os"
	"time"
)

var photoCache = mustNewLru[string, *gotgbot.File](500)
var NoImage = errors.New("no image or image too small")
var RateLimited = errors.New("rate limited")

func getPhotoCache(bot *gotgbot.Bot, photo *gotgbot.PhotoSize) (*gotgbot.File, error) {
	if res, found := photoCache.Get(photo.FileUniqueId); found {
		log.Debugf("get image file %s from photoCache", photo.FileUniqueId)
		return res, nil
	}
	file, err := bot.GetFile(photo.FileId, nil)
	if err != nil {
		log.Warnf("Download file %s error.", photo.FileUniqueId)
		return nil, err
	}
	photoCache.Add(photo.FileUniqueId, file)
	return file, nil

}

var ocrCache = mustNewLru[string, *azure.OcrResult](500)
var ocrRateLimiter = rate.NewLimiter(rate.Every(3*time.Second), 1)

func ocrMsg(bot *gotgbot.Bot, file *gotgbot.PhotoSize) (string, error) {
	log.Debugf("begin ocrMsg, file uid = %s", file.FileUniqueId)
	r := ocrRateLimiter.Reserve()
	dur := r.Delay()
	if dur > 3*time.Minute {
		log.Warnf("dur = %s > 3 minute, too long to ocr file %s", dur, file.FileUniqueId)
		r.Cancel()
		return "", RateLimited
	}
	time.Sleep(dur)
	if res, found := ocrCache.Get(file.FileUniqueId); found {
		log.Debugf("get image ocr result %s from ocrCache", file.FileUniqueId)
		return res.Text(), nil
	}

	localFile, err := getPhotoCache(bot, file)
	if err != nil {
		log.Warnf("Download ocr file %s error.", file.FileUniqueId)
		return "", err
	}
	log.Debugf("start remote ocr file %s", localFile.FilePath)
	res, err := globalcfg.Ocr.OcrFile(localFile.FilePath)
	if err != nil {
		log.Warnf("ocr file over, err = %s", err)
		return "", err
	}
	log.Debugf("ocr file over, result = %v", res)
	content := res.Text()
	ocrCache.Add(file.FileUniqueId, res)
	return content, nil
}

var moderatorMsgCache = mustNewLru[string, *azure.ModeratorResult](500)
var moderatorRateLimiter = rate.NewLimiter(rate.Every(3*time.Second), 1)

func moderatorMsg(bot *gotgbot.Bot, file *gotgbot.PhotoSize) (*azure.ModeratorResult, error) {
	r := moderatorRateLimiter.Reserve()
	dur := r.Delay()
	if dur > 3*time.Minute {
		r.Cancel()
		return nil, RateLimited
	}
	time.Sleep(dur)
	if res, found := moderatorMsgCache.Get(file.FileUniqueId); found {
		log.Debugf("get image %s moderator result (%f, %f) from ocrCache",
			file.FileUniqueId, res.RacyClassificationScore, res.AdultClassificationScore)
		return res, nil
	}
	localFile, err := getPhotoCache(bot, file)
	if err != nil {
		log.Warnf("Download ocr file %s error.", file.FileUniqueId)
		return nil, err
	}
	fp, err := os.Open(localFile.FilePath)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	cfg, _, err := image.DecodeConfig(fp)
	if err != nil {
		return nil, err
	}
	if cfg.Width < 128 || cfg.Height < 128 {
		return nil, NoImage
	}
	log.Debugf("start ocr file %s", localFile.FilePath)
	res, err := globalcfg.Moderator.EvalFile(localFile.FilePath)
	if err != nil {
		return nil, err
	}
	log.Debugf("get image %s moderator result (%f, %f) from azure",
		file.FileUniqueId, res.RacyClassificationScore, res.AdultClassificationScore)
	moderatorMsgCache.Add(file.FileUniqueId, res)
	return res, nil
}
