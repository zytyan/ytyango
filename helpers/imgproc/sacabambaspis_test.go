package imgproc

import (
	"os"
	"strings"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/require"
)

func TestGenSacaImage(t *testing.T) {
	as := require.New(t)
	img, err := GenSacaImage("sacasacasacabambamsacabaspispissaca")
	as.Nil(err)
	as.NotNil(img)
	const filename = "sacabambaspis.jpg"
	f, err := os.Create(filename)
	as.NoError(err)
	err = imaging.Encode(f, img, imaging.JPEG)
	as.NoError(err)
	os.Remove(filename)
}

func TestGenSacaImageErr(t *testing.T) {
	as := require.New(t)
	img, err := GenSacaImage(strings.Repeat("saca", 500))
	var e *ErrTooLongSacaList
	as.ErrorAs(err, &e)
	as.Nil(img)
}
