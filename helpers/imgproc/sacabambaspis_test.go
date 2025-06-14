package imgproc

import (
	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestGenSacaImage(t *testing.T) {
	as := require.New(t)
	img := GenSacaImage("sacasacasacabambamsacabaspispissaca")
	as.NotNil(img)
	const filename = "sacabambaspis.jpg"
	f, err := os.Create(filename)
	as.NoError(err)
	err = imaging.Encode(f, img, imaging.JPEG)
	as.NoError(err)
	os.Remove(filename)
}
