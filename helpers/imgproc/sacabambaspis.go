package imgproc

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	"regexp"
	"strings"
	"sync"
)

//go:embed assets/sakabanbasupisu.png
var sacabambaspis []byte

var getSakabanImage = sync.OnceValue(func() *image.NRGBA {
	img, err := imaging.Decode(bytes.NewReader(sacabambaspis))
	if err != nil {
		panic(err)
	}
	return img.(*image.NRGBA)
})

var reSacabambaspis = regexp.MustCompile(`(?i)^((萨卡|薩卡|saca|saka|さか|サカ)|(班|bam|ban|ばん|バン)|(甲|basu?|ばす|バス)|(鱼|魚|pisu?|ぴす|ピス))+$`)

func MatchSacabambaspis(text string) bool {
	return reSacabambaspis.MatchString(text)
}

type sacabamType int

const (
	saca sacabamType = iota
	bam
	bas
	pis
)

type sacaMap struct {
	name string
	enum sacabamType
}

var sacaToEnum = []sacaMap{
	// saca
	{"萨卡", saca}, {"薩卡", saca}, {"saca", saca},
	{"saka", saca}, {"さか", saca}, {"サカ", saca},
	// bam
	{"班", bam}, {"bam", bam}, {"ban", bam},
	{"ばん", bam}, {"バン", bam},
	// bas
	{"甲", bas}, {"basu", bas}, {"bas", bas},
	{"ばす", bas}, {"バス", bas},
	// pis
	{"鱼", pis}, {"魚", pis}, {"pisu", pis},
	{"pis", pis}, {"ぴす", pis}, {"ピス", pis},
}

const maxSacaListLen = 100

var ErrNoSacaList = errors.New("no saca list")

type ErrTooLongSacaList struct {
	Limit int
	Len   int
}

func (e *ErrTooLongSacaList) Error() string {
	return fmt.Sprintf("saca list too long, %d > %d", e.Len, e.Limit)
}
func strToSacabamList(text string) ([]sacabamType, error) {
	var result []sacabamType
	text = strings.ToLower(text)
	for len(text) > 0 {
		matched := false
		for _, mapping := range sacaToEnum {
			if len(text) >= len(mapping.name) && text[:len(mapping.name)] == mapping.name {
				result = append(result, mapping.enum)
				text = text[len(mapping.name):]
				matched = true
				break
			}
		}
		if !matched {
			// 无法识别，终止
			return nil, ErrNoSacaList
		}
	}
	if len(result) > maxSacaListLen {
		return nil, &ErrTooLongSacaList{Limit: maxSacaListLen, Len: len(result)}
	}
	return result, nil
}

var sacabamBgColor = color.RGBA{R: 254, G: 143, B: 0, A: 255}

var sacaRect = image.Rect(5, 0, 5+235, 371)
var bamRect = image.Rect(239, 0, 216+239, 371)
var basRect = image.Rect(454, 0, 454+235, 371)
var pisRect = image.Rect(688, 0, 688+238, 371)

func getSacaRectByType(typ sacabamType) image.Rectangle {
	switch typ {
	case saca:
		return sacaRect
	case bam:
		return bamRect
	case bas:
		return basRect
	case pis:
		return pisRect
	default:
		panic(fmt.Sprintf("getSacaSubImgByType: unsupported sacabam type: %v", typ))
	}
}
func getSacaSubImgByType(typ sacabamType) *image.NRGBA {
	return getSakabanImage().SubImage(getSacaRectByType(typ)).(*image.NRGBA)
}

func GenSacaImage(text string) (*image.NRGBA, error) {
	sacaList, err := strToSacabamList(text)
	if len(sacaList) == 0 || err != nil {
		return nil, err
	}
	posList := make([]int, 0, len(sacaList))
	posList = append(posList, 0)
	idx := 1
	for idx < len(sacaList) {
		if sacaList[idx] == saca {
			switch sacaList[idx-1] {
			case saca:
				newPos := posList[idx-1] + getSacaRectByType(sacaList[idx-1]).Dx() - 50
				posList = append(posList, newPos)
			case bam, bas:
				newPos := posList[idx-1] + getSacaRectByType(sacaList[idx-1]).Dx() - 70
				posList = append(posList, newPos)
			case pis:
				newPos := posList[idx-1] + getSacaRectByType(sacaList[idx-1]).Dx() - 80
				posList = append(posList, newPos)
			}
		} else if sacaList[idx-1] == pis && sacaList[idx] == pis {
			newPos := posList[idx-1] + getSacaRectByType(sacaList[idx-1]).Dx() - 20
			posList = append(posList, newPos)
		} else {
			newPos := posList[idx-1] + getSacaRectByType(sacaList[idx-1]).Dx()
			posList = append(posList, newPos)
		}
		idx++
	}
	width := posList[len(posList)-1] + getSacaRectByType(sacaList[len(posList)-1]).Dx() + 100
	height := sacaRect.Max.Y
	baseX := 50
	background := imaging.New(width, height, sacabamBgColor)
	img := background
	for i, pos := range posList {
		subImg := getSacaSubImgByType(sacaList[i])
		img = imaging.Overlay(img, subImg, image.Point{X: baseX + pos, Y: 0}, 1.0)
	}
	return img, nil
}
