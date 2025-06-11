package bothttp

import "strconv"

type JsonInt64 int64

func (j *JsonInt64) UnmarshalJSON(bs []byte) error {
	if bs[0] == '"' && bs[len(bs)-1] == '"' {
		bs = bs[1 : len(bs)-1]
	}
	x, err := strconv.ParseUint(string(bs), 10, 64)
	if err != nil {
		return err
	}
	*j = JsonInt64(x)
	return nil
}

func (j *JsonInt64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatInt(int64(*j), 10) + `"`), nil
}
