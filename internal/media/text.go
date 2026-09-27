package media

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"unicode/utf16"
	"unicode/utf8"
)

var (
	utf8BOM    = []byte{0xef, 0xbb, 0xbf}
	utf16LEBOM = []byte{0xff, 0xfe}
	utf16BEBOM = []byte{0xfe, 0xff}
)

func ValidateTextFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	_, err = DecodeText(data)
	return err
}

func ReadTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return DecodeText(data)
}

func DecodeText(data []byte) (string, error) {
	switch {
	case bytes.HasPrefix(data, utf8BOM):
		return decodeUTF8(
			data[len(utf8BOM):],
		)

	case bytes.HasPrefix(data, utf16LEBOM):
		return decodeUTF16(
			data[len(utf16LEBOM):],
			binary.LittleEndian,
		)

	case bytes.HasPrefix(data, utf16BEBOM):
		return decodeUTF16(
			data[len(utf16BEBOM):],
			binary.BigEndian,
		)

	default:
		return decodeUTF8(data)
	}
}

func decodeUTF8(data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", fmt.Errorf(
			"text is not valid UTF-8",
		)
	}

	text := string(data)

	if containsNUL(text) {
		return "", fmt.Errorf(
			"text contains NUL characters",
		)
	}

	return text, nil
}

func decodeUTF16(
	data []byte,
	byteOrder binary.ByteOrder,
) (string, error) {
	if len(data)%2 != 0 {
		return "", fmt.Errorf(
			"UTF-16 text contains an incomplete code unit",
		)
	}

	codeUnits := make(
		[]uint16,
		len(data)/2,
	)

	for i := range codeUnits {
		offset := i * 2

		codeUnits[i] = byteOrder.Uint16(
			data[offset : offset+2],
		)
	}

	runes := make(
		[]rune,
		0,
		len(codeUnits),
	)

	for i := 0; i < len(codeUnits); i++ {
		codeUnit := codeUnits[i]

		switch {
		case codeUnit >= 0xd800 &&
			codeUnit <= 0xdbff:
			if i+1 >= len(codeUnits) {
				return "", fmt.Errorf(
					"UTF-16 text contains an incomplete surrogate pair",
				)
			}

			next := codeUnits[i+1]
			if next < 0xdc00 ||
				next > 0xdfff {
				return "", fmt.Errorf(
					"UTF-16 text contains an invalid surrogate pair",
				)
			}

			runes = append(
				runes,
				utf16.DecodeRune(
					rune(codeUnit),
					rune(next),
				),
			)

			i++

		case codeUnit >= 0xdc00 &&
			codeUnit <= 0xdfff:
			return "", fmt.Errorf(
				"UTF-16 text contains an unexpected low surrogate",
			)

		default:
			runes = append(
				runes,
				rune(codeUnit),
			)
		}
	}

	text := string(runes)

	if containsNUL(text) {
		return "", fmt.Errorf(
			"text contains NUL characters",
		)
	}

	return text, nil
}

func containsNUL(text string) bool {
	for _, value := range text {
		if value == 0 {
			return true
		}
	}

	return false
}
