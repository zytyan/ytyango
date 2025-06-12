//go:build !windows

package imgproc

import (
	"errors"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"image"
	"os"
)

const webpEnabled = true

func encodeToWebp(img image.Image, file string) error {
	if img == nil {
		return errors.New("nil image")
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	opt, err := encoder.NewLossyEncoderOptions(encoder.PresetPicture, 80)
	if err != nil {
		return err
	}
	return webp.Encode(f, img, opt)
}
