package bili

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

func pow(base, exponent uint64) uint64 {
	if exponent == 0 {
		return 1
	}
	result := uint64(1)
	for exponent > 0 {
		if exponent&1 == 1 {
			result *= base
		}
		base *= base
		exponent >>= 1
	}
	return result
}

var keyword = regexp.MustCompile(`(https?://)?((b23\.tv/\w+)|((((www|m)\.)?bilibili\.com/(video/)?(av\d+|BV[\da-zA-Z]+))/?\?[\w=&%+._()*$@!^\-]+))`)
var bv2AvConvKey = map[int32]int{
	'1': 13, '2': 12, '3': 46, '4': 31, '5': 43, '6': 18,
	'7': 40, '8': 28, '9': 5, 'A': 54, 'B': 20,
	'C': 15, 'D': 8, 'E': 39, 'F': 57, 'G': 45, 'H': 36,
	'J': 38, 'K': 51, 'L': 42, 'M': 49, 'N': 52,
	'P': 53, 'Q': 7, 'R': 4, 'S': 9, 'T': 50, 'U': 10,
	'V': 44, 'W': 34, 'X': 6, 'Y': 25, 'Z': 1,
	'a': 26, 'b': 29, 'c': 56, 'd': 3, 'e': 24, 'f': 0,
	'g': 47, 'h': 27, 'i': 22, 'j': 41, 'k': 16,
	'm': 11, 'n': 37, 'o': 2, 'p': 35, 'q': 21, 'r': 17,
	's': 33, 't': 30, 'u': 48, 'v': 23, 'w': 55,
	'x': 32, 'y': 14, 'z': 19}
var bvPosExp = []uint64{6, 2, 4, 8, 5, 9, 3, 7, 1, 0}
var NotBv = errors.New("not a bv str")
var NoLocation = errors.New("b23 has no location")
var regexBv = regexp.MustCompile(`/[Bb][Vv][0-9a-zA-Z]{5,12}`)
var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func bv2av(bv string) string {
	// 1. 去除Bv号前的"/Bv"字符
	bv = bv[3:]
	// 2. 将key对应的value存入一个切片
	bv2 := make([]uint64, len(bv))
	for i, char := range bv {
		bv2[i] = uint64(bv2AvConvKey[char])
	}
	// 3. 对切片中不同位置的数进行*58的x次方的操作
	for i := range bv2 {
		bv2[i] = bv2[i] * pow(58, bvPosExp[i])
	}
	// 5. 将和减去100618342136696320
	av := uint64(0)
	for _, val := range bv2 {
		av += val
	}
	av -= 100618342136696320
	// 6. 将sum 与177451812进行异或
	av ^= 177451812
	return fmt.Sprintf("/av%d", av)
}

//goland:noinspection GoUnusedExportedFunction
func Bv2av(bv string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(bv), "/regexBv") {
		return "", NotBv
	}
	return bv2av(bv), nil
}

var validBvParams = []string{"p", "start_progress", "t"}
var validMallParams = []string{"itemsId"}
var BvNotFount = errors.New("regexBv not found")

func BvLink2AvLink(link string) (string, error) {
	index := regexBv.FindStringIndex(link)
	if index == nil {
		return "", BvNotFount
	}
	start, end := index[0], index[1]
	bvStr := link[start:end]
	av := bv2av(bvStr)
	// replace string by index
	newLink := link[:start] + av + link[end:]
	return newLink, nil

}

func paramsFilter(params url.Values, validParams ...string) url.Values {
	newQuery := make(url.Values)
	for _, param := range validParams {
		if params.Has(param) {
			newQuery.Add(param, params.Get(param))
		}
	}
	return newQuery
}

func BilibiliCleanParams(biliUrl string) (string, error) {
	parsedUrl, err := url.Parse(biliUrl)
	if err != nil {
		return "", err
	}
	parsedUrl.Scheme = "https"
	oldQuery := parsedUrl.Query()
	newQuery := make(url.Values)
	switch parsedUrl.Host {
	case "mall.bilibili.com":
		newQuery = paramsFilter(oldQuery, validMallParams...)
	case "www.bilibili.com", "bilibili.com":
		newQuery = paramsFilter(oldQuery, validBvParams...)
	default:
		newQuery = paramsFilter(oldQuery)
	}
	parsedUrl.RawQuery = newQuery.Encode()
	return parsedUrl.String(), nil
}

func B23ToBilibili(url string) (string, error) {
	if strings.EqualFold(url[:7], "http://") {
		url = "https://" + url[7:]
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	location := resp.Header.Get("Location")
	if location == "" {
		return "", NoLocation
	}
	return BilibiliCleanParams(location)
}

const (
	NotLink = iota
	B23
	Bilibili
)

type TextWithLink struct {
	Text      string
	Converted *string
	LinkType  int
}

func (t *TextWithLink) NeedConvert() bool {
	switch t.LinkType {
	case B23:
		return true
	case Bilibili:
		goto PROCESS
	case NotLink:
		return false
	default:
		return false
	}
PROCESS:
	u, err := url.Parse(t.Text)
	if err != nil {
		return false
	}
	q := u.Query()
	for key := range q {
		if !slices.Contains(validBvParams, key) {
			return true
		}
	}
	return false
}
func (t *TextWithLink) convert() (string, error) {
	switch t.LinkType {
	case B23:
		return B23ToBilibili(t.Text)
	case Bilibili:
		return BilibiliCleanParams(t.Text)
	case NotLink:
		return t.Text, nil
	default:
		return "<unknown link type>", errors.New("unknown link type")
	}
}
func (t *TextWithLink) Convert() (*string, error) {
	if t.Converted != nil {
		return t.Converted, nil
	}
	empty := ""
	converted, err := t.convert()
	if err != nil {
		return &empty, err
	}
	t.Converted = &converted
	return &converted, nil
}
func (t *TextWithLink) ConvertToAv() (string, error) {
	if t.Converted == nil {
		_, err := t.Convert()
		if err != nil {
			return "", err
		}
	}
	link, err := BvLink2AvLink(*t.Converted)
	if err != nil {
		// when regexBv not found, just return the original link
		return *t.Converted, nil
	}
	return link, nil
}

type ContentWithLinks []TextWithLink

func (c *ContentWithLinks) NeedConvert() bool {
	for _, link := range *c {
		if link.NeedConvert() {
			return true
		}
	}
	return false
}
func (c *ContentWithLinks) ToBv() (string, error) {
	buf := make([]string, 0)
	for _, link := range *c {
		bv, err := link.Convert()
		if err != nil {
			return "", err
		}
		buf = append(buf, *bv)
	}
	return strings.Join(buf, ""), nil
}
func (c *ContentWithLinks) ToAv() (string, error) {
	buf := make([]string, 0)
	for _, link := range *c {
		av, err := link.ConvertToAv()
		if err != nil {
			return "", err
		}
		buf = append(buf, av)
	}
	return strings.Join(buf, ""), nil
}

var reBvRegex = regexp.MustCompile(`[Bb][Vv][0-9a-zA-Z]{5,12}`)

func (c *ContentWithLinks) FirstBvId() (string, error) {
	for _, link := range *c {
		if link.LinkType != NotLink {
			v, err := link.Convert()
			if err != nil {
				return "", err
			}
			if reBvRegex.FindString(*v) != "" {
				return *v, nil
			}
		}
	}
	return "", errors.New("no Bilibili link")
}

func ContainsBiliLinkAndTryPrepare(text string) (links ContentWithLinks, err error) {
	indexList := keyword.FindAllStringIndex(text, -1)
	if indexList == nil {
		return nil, errors.New("no Bilibili links")
	}
	links = make([]TextWithLink, 0)
	lastIndex := 0
	for _, index := range indexList {
		start, end := index[0], index[1]
		if start > lastIndex {
			links = append(links, TextWithLink{Text: text[lastIndex:start], LinkType: NotLink})
		}
		link := text[start:end]
		lowerLink := strings.ToLower(link)
		if strings.Contains(lowerLink, "b23.tv") {
			links = append(links, TextWithLink{Text: link, LinkType: B23})
		} else {
			links = append(links, TextWithLink{Text: link, LinkType: Bilibili})
		}
		lastIndex = end
	}
	if lastIndex < len(text) {
		links = append(links, TextWithLink{Text: text[lastIndex:], LinkType: NotLink})
	}
	return links, nil
}
