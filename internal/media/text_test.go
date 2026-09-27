package media

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestDecodeTextUTF8(
	t *testing.T,
) {
	input := []byte(
		"Hallo Welt äöü €",
	)

	text, err := DecodeText(input)
	if err != nil {
		t.Fatalf(
			"decode UTF-8: %v",
			err,
		)
	}

	if text != string(input) {
		t.Fatalf(
			"unexpected text: %q",
			text,
		)
	}
}

func TestDecodeTextUTF8BOM(
	t *testing.T,
) {
	input := append(
		[]byte{
			0xef,
			0xbb,
			0xbf,
		},
		[]byte("Hallo Welt")...,
	)

	text, err := DecodeText(input)
	if err != nil {
		t.Fatalf(
			"decode UTF-8 BOM: %v",
			err,
		)
	}

	if text != "Hallo Welt" {
		t.Fatalf(
			"unexpected text: %q",
			text,
		)
	}
}

func TestDecodeTextUTF16LittleEndian(
	t *testing.T,
) {
	input := encodeUTF16TestData(
		"Hallo Welt äöü 😀",
		binary.LittleEndian,
		[]byte{
			0xff,
			0xfe,
		},
	)

	text, err := DecodeText(input)
	if err != nil {
		t.Fatalf(
			"decode UTF-16 LE: %v",
			err,
		)
	}

	if text != "Hallo Welt äöü 😀" {
		t.Fatalf(
			"unexpected text: %q",
			text,
		)
	}
}

func TestDecodeTextUTF16BigEndian(
	t *testing.T,
) {
	input := encodeUTF16TestData(
		"Hallo Welt äöü 😀",
		binary.BigEndian,
		[]byte{
			0xfe,
			0xff,
		},
	)

	text, err := DecodeText(input)
	if err != nil {
		t.Fatalf(
			"decode UTF-16 BE: %v",
			err,
		)
	}

	if text != "Hallo Welt äöü 😀" {
		t.Fatalf(
			"unexpected text: %q",
			text,
		)
	}
}

func TestDecodeTextRejectsInvalidUTF8(
	t *testing.T,
) {
	_, err := DecodeText(
		[]byte{
			0xff,
			0xff,
			0xff,
		},
	)

	if err == nil {
		t.Fatal(
			"expected invalid UTF-8 to be rejected",
		)
	}
}

func TestDecodeTextRejectsNUL(
	t *testing.T,
) {
	_, err := DecodeText(
		[]byte{
			'a',
			0x00,
			'b',
		},
	)

	if err == nil {
		t.Fatal(
			"expected NUL characters to be rejected",
		)
	}
}

func TestDecodeTextRejectsIncompleteUTF16(
	t *testing.T,
) {
	_, err := DecodeText(
		[]byte{
			0xff,
			0xfe,
			0x41,
		},
	)

	if err == nil {
		t.Fatal(
			"expected incomplete UTF-16 to be rejected",
		)
	}
}

func TestDecodeTextRejectsInvalidUTF16Surrogate(
	t *testing.T,
) {
	input := []byte{
		0xff,
		0xfe,
		0x00,
		0xd8,
		0x41,
		0x00,
	}

	_, err := DecodeText(input)

	if err == nil {
		t.Fatal(
			"expected invalid UTF-16 surrogate pair to be rejected",
		)
	}
}

func encodeUTF16TestData(
	text string,
	byteOrder binary.ByteOrder,
	bom []byte,
) []byte {
	codeUnits := utf16.Encode(
		[]rune(text),
	)

	data := make(
		[]byte,
		len(bom)+len(codeUnits)*2,
	)

	copy(
		data,
		bom,
	)

	offset := len(bom)

	for _, codeUnit := range codeUnits {
		byteOrder.PutUint16(
			data[offset:offset+2],
			codeUnit,
		)

		offset += 2
	}

	return data
}
