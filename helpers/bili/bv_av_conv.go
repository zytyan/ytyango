package bili

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var reAv = regexp.MustCompile(`(?i)/av\d+`)
var reBv = regexp.MustCompile(`(?i)/BV[\da-zA-Z]+`)
var reAvOrBv = regexp.MustCompile(`(?i)(/av\d+|/BV[\da-zA-Z]+)`)

const (
	XorCode = 23442827791579
	BASE    = 58
	TABLE   = "FcwAPNKTMug3GV5Lj7EJnHpWsx4tb8haYeviqBz6rkCy12mUSDQX9RdoZf"
)

var reverseTable = func() [128]byte {
	var table [128]byte
	for i := range table {
		table[i] = 0xFF
	}
	for i, ch := range TABLE {
		table[ch] = byte(i)
	}
	return table

}()

func bv2av(bvid string) string {
	if len(bvid) != 13 || !strings.HasPrefix(bvid, "/BV1") {
		return bvid
	}

	bytes := []byte(bvid[1:])

	// 交换索引 3 和 9 的字符
	bytes[3], bytes[9] = bytes[9], bytes[3]
	// 交换索引 4 和 7 的字符
	bytes[4], bytes[7] = bytes[7], bytes[4]

	bvid = string(bytes[3:]) // 删除前3个字符 "BV1"
	tmp := int64(0)

	for _, ch := range bvid {
		idx := reverseTable[ch]
		if idx == 0xFF {
			return bvid
		}
		tmp = tmp*BASE + int64(idx)
	}

	avid := (tmp & ((1 << 51) - 1)) ^ XorCode
	return "/av" + strconv.FormatInt(avid, 10)
}

func av2bv(av string) string {
	av = av[3:] // 去除 "/av" 前缀
	aid, err := strconv.ParseInt(av, 10, 64)
	if err != nil {
		panic(err)
	}
	bytes := []byte{'B', 'V', '1', '0', '0', '0', '0', '0', '0', '0', '0', '0'}
	bvIdx := len(bytes) - 1
	tmp := (aid | (1 << 51)) ^ XorCode

	for tmp > 0 {
		bytes[bvIdx] = TABLE[tmp%BASE]
		tmp /= BASE
		bvIdx--
	}

	// 交换索引 3 和 9 的元素
	bytes[3], bytes[9] = bytes[9], bytes[3]
	// 交换索引 4 和 7 的元素
	bytes[4], bytes[7] = bytes[7], bytes[4]

	return "/" + string(bytes)
}

var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func followRedirects(url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()
	return resp.Header.Get("Location"), nil
}

func HasVideoLink(link string) bool {
	return reAvOrBv.MatchString(link)
}
