package converter

import (
	"fmt"
	"image"
	"os"
	"regexp"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var pdfTextColorPattern = regexp.MustCompile(
	`^#[0-9a-fA-F]{6}$`,
)

func textWatermark(
	edit PDFTextEdit,
	dimensions []types.Dim,
) (*model.Watermark, error) {
	if err := validatePDFTextEdit(
		edit,
		len(dimensions),
	); err != nil {
		return nil, err
	}

	dimension :=
		dimensions[edit.Page-1]

	x :=
		edit.X *
			dimension.Width

	y :=
		dimension.Height -
			edit.Y*
				dimension.Height -
			float64(edit.Size)

	if y < 0 {
		y = 0
	}

	description :=
		fmt.Sprintf(
			"font:Helvetica, "+
				"points:%d, "+
				"scale:1 abs, "+
				"pos:bl, "+
				"off:%.4f %.4f, "+
				"align:l, "+
				"fillc:%s, "+
				"rot:0, "+
				"op:1, "+
				"rendermode:0",
			edit.Size,
			x,
			y,
			edit.Color,
		)

	watermark, err :=
		api.TextWatermark(
			edit.Text,
			description,
			true,
			false,
			types.POINTS,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create text stamp for page %d: %w",
				edit.Page,
				err,
			)
	}

	return watermark, nil
}

func imageWatermark(
	edit PDFImageEdit,
	dimensions []types.Dim,
) (*model.Watermark, error) {
	if err := validatePDFImageEdit(
		edit,
		len(dimensions),
	); err != nil {
		return nil, err
	}

	config, err :=
		readPDFEditImageConfig(
			edit.Path,
		)
	if err != nil {
		return nil, err
	}

	dimension :=
		dimensions[edit.Page-1]

	targetWidth :=
		edit.Width *
			dimension.Width

	scale :=
		targetWidth /
			float64(config.Width)

	targetHeight :=
		float64(config.Height) *
			scale

	x :=
		edit.X *
			dimension.Width

	y :=
		dimension.Height -
			edit.Y*
				dimension.Height -
			targetHeight

	if y < 0 {
		y = 0
	}

	description :=
		fmt.Sprintf(
			"scale:%.8f abs, "+
				"pos:bl, "+
				"off:%.4f %.4f, "+
				"rot:0, "+
				"op:1",
			scale,
			x,
			y,
		)

	watermark, err :=
		api.ImageWatermark(
			edit.Path,
			description,
			true,
			false,
			types.POINTS,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create image stamp for page %d: %w",
				edit.Page,
				err,
			)
	}

	return watermark, nil
}

func validatePDFTextEdit(
	edit PDFTextEdit,
	pageCount int,
) error {
	if edit.Page < 1 ||
		edit.Page > pageCount {
		return fmt.Errorf(
			"invalid page number %d",
			edit.Page,
		)
	}

	if strings.TrimSpace(
		edit.Text,
	) == "" {
		return fmt.Errorf(
			"text on page %d is empty",
			edit.Page,
		)
	}

	if len(
		[]rune(edit.Text),
	) > maxPDFTextLength {
		return fmt.Errorf(
			"text on page %d exceeds maximum length",
			edit.Page,
		)
	}

	if edit.X < 0 ||
		edit.X > 1 {
		return fmt.Errorf(
			"invalid x position on page %d",
			edit.Page,
		)
	}

	if edit.Y < 0 ||
		edit.Y > 1 {
		return fmt.Errorf(
			"invalid y position on page %d",
			edit.Page,
		)
	}

	if edit.Size < 6 ||
		edit.Size > 144 {
		return fmt.Errorf(
			"invalid font size on page %d",
			edit.Page,
		)
	}

	if !pdfTextColorPattern.MatchString(
		edit.Color,
	) {
		return fmt.Errorf(
			"invalid text color on page %d",
			edit.Page,
		)
	}

	return nil
}

func validatePDFImageEdit(
	edit PDFImageEdit,
	pageCount int,
) error {
	if edit.Page < 1 ||
		edit.Page > pageCount {
		return fmt.Errorf(
			"invalid image page number %d",
			edit.Page,
		)
	}

	if edit.Path == "" {
		return fmt.Errorf(
			"image path is empty",
		)
	}

	if edit.X < 0 ||
		edit.X > 1 {
		return fmt.Errorf(
			"invalid image x position on page %d",
			edit.Page,
		)
	}

	if edit.Y < 0 ||
		edit.Y > 1 {
		return fmt.Errorf(
			"invalid image y position on page %d",
			edit.Page,
		)
	}

	if edit.Width < 0.05 ||
		edit.Width > 1 {
		return fmt.Errorf(
			"invalid image width on page %d",
			edit.Page,
		)
	}

	return nil
}

func readPDFEditImageConfig(
	path string,
) (image.Config, error) {
	file, err := os.Open(
		path,
	)
	if err != nil {
		return image.Config{},
			fmt.Errorf(
				"open PDF edit image: %w",
				err,
			)
	}
	defer file.Close()

	config, format, err :=
		image.DecodeConfig(
			file,
		)
	if err != nil {
		return image.Config{},
			fmt.Errorf(
				"decode PDF edit image: %w",
				err,
			)
	}

	switch format {
	case "jpeg",
		"png":
	default:
		return image.Config{},
			fmt.Errorf(
				"unsupported PDF edit image format %q",
				format,
			)
	}

	if config.Width <= 0 ||
		config.Height <= 0 {
		return image.Config{},
			fmt.Errorf(
				"invalid PDF edit image dimensions",
			)
	}

	return config, nil
}
