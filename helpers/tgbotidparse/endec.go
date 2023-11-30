package tgbotidparse

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
)

func rleEncode(data []byte) []byte {
	var result []byte
	// 当某字节为 0时，第二个字节表示该字节重复的次数
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			var count byte
			for j := i; j < len(data); j++ {
				if data[j] == 0 {
					count++
				} else {
					break
				}
			}
			result = append(result, 0, count)
			i += int(count) - 1
		} else {
			result = append(result, data[i])
		}
	}
	return result
}
func rleDecode(data []byte) []byte {
	var result []byte
	// 当某字节为 0时，第二个字节表示该字节重复的次数
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			for j := 0; j < int(data[i+1]); j++ {
				result = append(result, 0)
			}
			i++
		} else {
			result = append(result, data[i])
		}
	}
	return result
}

func packTLString(input []byte) []byte {
	length := len(input)
	var concat []byte
	var fill int
	if length <= 253 {
		concat = append(concat, byte(length))
		fill = posMod(-length-1, 4)
	} else {
		concat = append(concat, 254)
		lengthBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(lengthBytes, uint32(length))
		concat = append(concat, lengthBytes[:3]...)
		fill = posMod(-length, 4)
	}
	concat = append(concat, input...)
	fillBytes := make([]byte, fill)
	concat = append(concat, fillBytes...)

	return concat
}
func unpackTLString(data []byte) ([]byte, error, int) {
	if len(data) == 0 {
		return nil, errors.New("empty data buffer"), 0
	}
	offset := 0
	lengthByte := data[0]
	length := int(lengthByte)

	if length > 254 {
		return nil, errors.New("length too big for a single field"), 0
	}
	// Untested: Read the next 3 bytes as little-endian uint32
	data = data[1:]
	offset++
	if len(data) < 3 {
		return nil, errors.New("insufficient data for length"), 0
	}
	var fill int
	if length == 254 {
		lenBuf := make([]byte, 4)
		copy(lenBuf, data[:3])
		length = int(binary.LittleEndian.Uint32(lenBuf))
		data = data[3:]
		offset += 3
		fill = posMod(-length, 4)
	} else {
		fill = posMod(-length-1, 4)
	}
	if len(data) < length {
		return nil, errors.New("insufficient data for string"), 0
	}
	// Read the string bytes
	if len(data) < fill {
		return nil, errors.New("insufficient data for fill"), 0
	}
	return data[:length], nil, offset + length + fill
}
func posMod(a, b int) int {
	rest := a % b
	if rest < 0 {
		if b < 0 {
			b = -b
		}
		return rest + b
	}
	return rest
}
func decodeBotFileId(id string) ([]byte, error) {
	decode, err := base64.URLEncoding.DecodeString(id)
	if err != nil {
		return nil, err
	}
	return rleDecode(decode), nil
}
