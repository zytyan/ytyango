package imgproc

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"sync"

	"github.com/disintegration/imaging"
)

//go:embed assets/prpr_base.png
var basePrpr []byte

var getBasePrpr = sync.OnceValue(func() *image.NRGBA {
	base, err := imaging.Decode(bytes.NewReader(basePrpr))
	if err != nil {
		panic(err)
	}
	return base.(*image.NRGBA)
})

func prprBase(profilePhoto string) image.Image {
	const W, H = 143, 143
	const L, T = 306, 362
	f2, err := imaging.Open(profilePhoto)
	if err != nil {
		panic(err)
	}
	f2 = imaging.Resize(f2, W, H, imaging.Lanczos)
	rect := getBasePrpr().Rect
	empty := imaging.New(rect.Dx(), rect.Dy(), image.Transparent)
	out := imaging.Overlay(empty, f2, image.Pt(L, T), 1.0)
	out = imaging.Overlay(out, getBasePrpr(), image.Pt(0, 0), 1.0)
	return out
}

func GenPrpr(profilePhoto string, file string) error {
	return encodeToWebp(prprBase(profilePhoto), file)
}
