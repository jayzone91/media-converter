package media

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const signatureReadSize = 4096

func DetectSignature(
	path string,
) (Format, error) {
	file, err := os.Open(path)
	if err != nil {
		return Format{}, err
	}
	defer file.Close()

	buffer := make(
		[]byte,
		signatureReadSize,
	)

	n, err := file.Read(buffer)
	if err != nil &&
		err != io.EOF {
		return Format{}, err
	}

	data := buffer[:n]

	if format, ok := detectBinarySignature(
		data,
	); ok {
		return format, nil
	}

	if isSVG(data) {
		return Formats["svg"], nil
	}

	return Format{},
		fmt.Errorf(
			"no supported file signature detected",
		)
}

func detectBinarySignature(
	data []byte,
) (Format, bool) {
	switch {
	case hasPrefix(
		data,
		[]byte{
			0x89,
			0x50,
			0x4e,
			0x47,
			0x0d,
			0x0a,
			0x1a,
			0x0a,
		},
	):
		return Formats["png"], true

	case hasPrefix(
		data,
		[]byte{
			0xff,
			0xd8,
			0xff,
		},
	):
		return Formats["jpeg"], true

	case isWebP(data):
		return Formats["webp"], true

	case hasPrefix(
		data,
		[]byte("GIF87a"),
	):
		return Formats["gif"], true

	case hasPrefix(
		data,
		[]byte("GIF89a"),
	):
		return Formats["gif"], true

	case hasPrefix(
		data,
		[]byte{
			0x42,
			0x4d,
		},
	):
		return Formats["bmp"], true

	case isTIFF(data):
		return Formats["tiff"], true

	case hasPrefix(
		data,
		[]byte("%PDF-"),
	):
		return Formats["pdf"], true
	}

	if format, ok := detectISOBaseMedia(
		data,
	); ok {
		return format, true
	}

	return Format{}, false
}

func hasPrefix(
	data []byte,
	signature []byte,
) bool {
	return len(data) >= len(signature) &&
		bytes.Equal(
			data[:len(signature)],
			signature,
		)
}

func isWebP(
	data []byte,
) bool {
	return len(data) >= 12 &&
		bytes.Equal(
			data[0:4],
			[]byte("RIFF"),
		) &&
		bytes.Equal(
			data[8:12],
			[]byte("WEBP"),
		)
}

func isTIFF(
	data []byte,
) bool {
	if len(data) < 4 {
		return false
	}

	littleEndian :=
		bytes.Equal(
			data[:4],
			[]byte{
				0x49,
				0x49,
				0x2a,
				0x00,
			},
		)

	bigEndian :=
		bytes.Equal(
			data[:4],
			[]byte{
				0x4d,
				0x4d,
				0x00,
				0x2a,
			},
		)

	return littleEndian ||
		bigEndian
}

func detectISOBaseMedia(
	data []byte,
) (Format, bool) {
	if len(data) < 12 ||
		!bytes.Equal(
			data[4:8],
			[]byte("ftyp"),
		) {
		return Format{}, false
	}

	brands := collectISOBrands(
		data,
	)

	if containsBrand(
		brands,
		"avif",
		"avis",
	) {
		return Formats["avif"], true
	}

	if containsBrand(
		brands,
		"heic",
		"heix",
		"hevc",
		"hevx",
		"heim",
		"heis",
		"hevm",
		"hevs",
	) {
		return Formats["heic"], true
	}

	return Format{}, false
}

func collectISOBrands(
	data []byte,
) []string {
	if len(data) < 12 {
		return nil
	}

	brands := []string{
		string(data[8:12]),
	}

	for offset := 16; offset+4 <= len(data); offset += 4 {
		brands = append(
			brands,
			string(
				data[offset:offset+4],
			),
		)
	}

	return brands
}

func containsBrand(
	brands []string,
	candidates ...string,
) bool {
	for _, brand := range brands {
		for _, candidate := range candidates {
			if brand == candidate {
				return true
			}
		}
	}

	return false
}

func isSVG(
	data []byte,
) bool {
	if len(data) == 0 {
		return false
	}

	data = bytes.TrimPrefix(
		data,
		[]byte{
			0xef,
			0xbb,
			0xbf,
		},
	)

	data = bytes.TrimSpace(
		data,
	)

	lower := bytes.ToLower(
		data,
	)

	if bytes.HasPrefix(
		lower,
		[]byte("<svg"),
	) {
		return true
	}

	if !bytes.HasPrefix(
		lower,
		[]byte("<?xml"),
	) {
		return false
	}

	return bytes.Contains(
		lower,
		[]byte("<svg"),
	)
}
