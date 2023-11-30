//go:build windows

package imgproc

import (
	"errors"
	"image"
)

func encodeToWebp(_ image.Image, _ string) error {
	return errors.New("webp not support on windows")
}
