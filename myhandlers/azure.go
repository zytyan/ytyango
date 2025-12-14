package myhandlers

import (
	"errors"
	"image"
	"main/globalcfg"
	"main/helpers/azure"
	"math/rand"
	"os"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"golang.org/x/time/rate"
)

var photoCache = mustNewLru[string, *gotgbot.File](500)
var NoImage = errors.New("no image or image too small")
var RateLimited = errors.New("rate limited")

type limiterConfig struct {
	rpm         int
	minInterval time.Duration
}

const (
	maxReservationDelay = 3 * time.Minute
	maxRetryCount       = 3
	backoffBaseDelay    = 500 * time.Millisecond
)

var (
	ocrLimiterConfig       = limiterConfig{rpm: 20, minInterval: 3 * time.Second}
	moderatorLimiterConfig = limiterConfig{rpm: 20, minInterval: 3 * time.Second}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func newRateLimiter(cfg limiterConfig) *rate.Limiter {
	if cfg.rpm <= 0 {
		cfg.rpm = 1
	}
	interval := time.Minute / time.Duration(cfg.rpm)
	if interval < cfg.minInterval {
		interval = cfg.minInterval
	}
	return rate.NewLimiter(rate.Every(interval), 1)
}

func waitForTurn(l *rate.Limiter, key string) error {
	r := l.Reserve()
	dur := r.Delay()
	if dur > maxReservationDelay {
		log.Warnf("dur = %s > %s, too long to handle %s", dur, maxReservationDelay, key)
		r.Cancel()
		return RateLimited
	}
	time.Sleep(dur)
	return nil
}

func jitteredBackoff(attempt int) time.Duration {
	delay := backoffBaseDelay << attempt
	if delay <= 0 {
		delay = backoffBaseDelay
	}
	jitter := time.Duration(rand.Int63n(int64(delay)/2 + 1))
	return delay + jitter
}

func withRetry[T any](op func() (T, error)) (T, error) {
	var zero T
	for attempt := 0; attempt < maxRetryCount; attempt++ {
		res, err := op()
		if err == nil {
			return res, nil
		}
		if attempt == maxRetryCount-1 {
			return zero, err
		}
		time.Sleep(jitteredBackoff(attempt))
	}
	return zero, nil
}

func getPhotoCache(bot *gotgbot.Bot, photo *gotgbot.PhotoSize) (*gotgbot.File, error) {
	if res, found := photoCache.Get(photo.FileUniqueId); found {
		log.Debugf("get image file %s from photoCache", photo.FileUniqueId)
		return res, nil
	}
	file, err := bot.GetFile(photo.FileId, nil)
	if err != nil {
		log.Warnf("YtDownloadResult file %s error.", photo.FileUniqueId)
		return nil, err
	}
	photoCache.Add(photo.FileUniqueId, file)
	return file, nil

}

var ocrCache = mustNewLru[string, *azure.OcrResult](500)
var ocrRateLimiter = newRateLimiter(ocrLimiterConfig)

func ocrMsg(bot *gotgbot.Bot, file *gotgbot.PhotoSize) (string, error) {
	log.Debugf("begin ocrMsg, file uid = %s", file.FileUniqueId)
	if err := waitForTurn(ocrRateLimiter, file.FileUniqueId); err != nil {
		return "", err
	}
	if res, found := ocrCache.Get(file.FileUniqueId); found {
		log.Debugf("get image ocr result %s from ocrCache", file.FileUniqueId)
		return res.Text(), nil
	}

	localFile, err := getPhotoCache(bot, file)
	if err != nil {
		log.Warnf("YtDownloadResult ocr file %s error.", file.FileUniqueId)
		return "", err
	}
	log.Debugf("start remote ocr file %s", localFile.FilePath)
	res, err := withRetry(func() (*azure.OcrResult, error) {
		return g.Ocr.OcrFile(localFile.FilePath)
	})
	if err != nil {
		log.Warnf("ocr file over, err = %s", err)
		return "", err
	}
	log.Debugf("ocr file over, result = %v", res)
	content := res.Text()
	ocrCache.Add(file.FileUniqueId, res)
	return content, nil
}

var moderatorMsgCache = mustNewLru[string, *azure.ModeratorV2Result](500)
var moderatorRateLimiter = newRateLimiter(moderatorLimiterConfig)

func moderatorMsg(bot *gotgbot.Bot, file *gotgbot.PhotoSize) (*azure.ModeratorV2Result, error) {
	if err := waitForTurn(moderatorRateLimiter, file.FileUniqueId); err != nil {
		return nil, err
	}
	if res, found := moderatorMsgCache.Get(file.FileUniqueId); found {
		log.Debugf("get image %s moderator result sexual severity %d",
			file.FileUniqueId, res.GetSeverityByCategory(azure.ModerateV2CatSexual))
		return res, nil
	}
	localFile, err := getPhotoCache(bot, file)
	if err != nil {
		log.Warnf("YtDownloadResult ocr file %s error.", file.FileUniqueId)
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
	res, err := withRetry(func() (*azure.ModeratorV2Result, error) {
		return g.Moderator.EvalFile(localFile.FilePath)
	})
	if err != nil {
		return nil, err
	}
	log.Debugf("get image %s moderator result sexual severity %d",
		file.FileUniqueId, res.GetSeverityByCategory(azure.ModerateV2CatSexual))
	moderatorMsgCache.Add(file.FileUniqueId, res)
	return res, nil
}
