package bili

import (
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var keyword = regexp.MustCompile(`(?i)(https?://)?\w+(\.(\w)+)+/[\w#?/=&%+._()*$@!^\-]+`)
var httpSchema = regexp.MustCompile(`^https?://`)

var validBvParams = []string{"p", "start_progress", "t"}
var validMallParams = []string{"itemsId"}

func cleanParams(u *url.URL, validParams []string) (string, bool) {
	q := u.Query()
	res := false
	for k := range q {
		if !slices.Contains(validParams, k) {
			res = true
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), res

}

type linkConverted struct {
	avLink string
	bvLink string // 默认为BV链接，即使为非视频，也认为是BV链接
	hasAv  bool
	hasBv  bool

	needClean bool
}

func tryParseLink(s string) (l linkConverted) {
	l = linkConverted{
		avLink:    s,
		bvLink:    s,
		hasAv:     false,
		hasBv:     false,
		needClean: false,
	}
	if !httpSchema.MatchString(s) {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return
	}
	u.Scheme = "https"
	if u.Host == "b23.tv" || u.Host == "bili2233.cn" {
		follow, err := followRedirects(u.String())
		if err != nil {
			return
		}
		if follow == "" {
			return
		}
		u, err = url.Parse(follow)
		if err != nil {
			return
		}
	}
	switch u.Host {
	case "bilibili.com", "www.bilibili.com", "m.bilibili.com":
		if !reAvOrBv.MatchString(u.Path) {
			link, ok := cleanParams(u, validBvParams)
			l.bvLink = link
			l.hasBv = true
			l.needClean = ok
			return
		}
		link, ok := cleanParams(u, validBvParams)
		l.needClean = ok
		if reBv.MatchString(link) {
			l.bvLink = link
			l.hasBv = true
			hasAv := false
			l.avLink = reBv.ReplaceAllStringFunc(link, func(s string) string {
				hasAv = true
				return bv2av(s)
			})
			l.hasAv = hasAv
		} else if reAv.MatchString(link) {
			l.avLink = link
			l.hasAv = true
			hasBv := false
			l.bvLink = reAv.ReplaceAllStringFunc(link, func(s string) string {
				hasBv = true
				return av2bv(s)
			})
			l.hasBv = hasBv
		}
		return
	case "mall.bilibili.com":
		l.bvLink, l.needClean = cleanParams(u, validMallParams)
		l.hasBv = true
		return
	default:
		return
	}
}

type Converted struct {
	Raw    string
	AvText string
	BvText string
	HasAv  bool
	HasBv  bool

	NeedClean bool
}

// links bilibili.com b23.tv bili2233.cn
var reFastCheck = regexp.MustCompile(`(?i)bilibili\.com|b23\.tv|bili2233\.cn`)

func fastCheck(text string) bool {
	return reFastCheck.MatchString(text)
}

func ConvertBilibiliLinks(text string) (c Converted, err error) {
	if !fastCheck(text) {
		return c, errors.New("no Bilibili links")
	}
	indexList := keyword.FindAllStringIndex(text, -1)
	if indexList == nil {
		return c, errors.New("no Bilibili links")
	}
	lastIndex := 0

	avBuf := strings.Builder{}
	bvBuf := strings.Builder{}
	avBuf.Grow(len(text) * 2)
	bvBuf.Grow(len(text) * 2)

	for _, index := range indexList {
		start, end := index[0], index[1]
		if start > lastIndex {
			avBuf.WriteString(text[lastIndex:start])
			bvBuf.WriteString(text[lastIndex:start])
		}
		link := text[start:end]
		parsed := tryParseLink(link)
		c.HasAv = parsed.hasAv || c.HasAv
		c.HasBv = parsed.hasBv || c.HasBv
		c.NeedClean = parsed.needClean || c.NeedClean
		avBuf.WriteString(parsed.avLink)
		bvBuf.WriteString(parsed.bvLink)
		lastIndex = end
	}
	if lastIndex < len(text) {
		avBuf.WriteString(text[lastIndex:])
		bvBuf.WriteString(text[lastIndex:])
	}
	c.Raw = text
	c.AvText = avBuf.String()
	c.BvText = bvBuf.String()
	return c, nil
}

func (c *Converted) CanConvert() bool {
	return c.HasAv || c.HasBv
}
