package web

import (
	"reflect"
	"testing"

	"github.com/jayzone91/media-converter/internal/media"
)

func TestConversionWorkloads(
	t *testing.T,
) {
	t.Parallel()

	tests :=
		[]struct {
			name string

			format media.Format
			target string

			workloads []workloadType
		}{
			{
				name: "image uses ImageMagick",

				format: media.Formats["png"],

				target: "jpeg",

				workloads: []workloadType{
					workloadImageMagick,
				},
			},
			{
				name: "gif to video uses FFmpeg",

				format: media.Formats["gif"],

				target: "mp4",

				workloads: []workloadType{
					workloadFFmpeg,
				},
			},
			{
				name: "audio uses FFmpeg",

				format: media.Formats["mp3"],

				target: "wav",

				workloads: []workloadType{
					workloadFFmpeg,
				},
			},
			{
				name: "video uses FFmpeg",

				format: media.Formats["mp4"],

				target: "webm",

				workloads: []workloadType{
					workloadFFmpeg,
				},
			},
			{
				name: "document uses LibreOffice",

				format: media.Formats["docx"],

				target: "pdf",

				workloads: []workloadType{
					workloadLibreOffice,
				},
			},
			{
				name: "txt to html needs no external workload",

				format: media.Formats["txt"],

				target: "html",
			},
			{
				name: "markdown to html needs no external workload",

				format: media.Formats["markdown"],

				target: "html",
			},
			{
				name: "markdown to pdf uses Chromium",

				format: media.Formats["markdown"],

				target: "pdf",

				workloads: []workloadType{
					workloadChromium,
				},
			},
			{
				name: "markdown to image uses Chromium and ImageMagick",

				format: media.Formats["markdown"],

				target: "png",

				workloads: []workloadType{
					workloadChromium,
					workloadImageMagick,
				},
			},
			{
				name: "pdf to docx uses LibreOffice and Poppler",

				format: media.Format{
					ID: "pdf",

					Category: media.CategoryPDF,
				},

				target: "docx",

				workloads: []workloadType{
					workloadLibreOffice,
					workloadPoppler,
				},
			},
			{
				name: "pdf to png uses Poppler",

				format: media.Format{
					ID: "pdf",

					Category: media.CategoryPDF,
				},

				target: "png",

				workloads: []workloadType{
					workloadPoppler,
				},
			},
			{
				name: "pdf to jpeg uses Poppler",

				format: media.Format{
					ID: "pdf",

					Category: media.CategoryPDF,
				},

				target: "jpeg",

				workloads: []workloadType{
					workloadPoppler,
				},
			},
		}

	for _, test := range tests {
		test :=
			test

		t.Run(
			test.name,
			func(
				t *testing.T,
			) {
				t.Parallel()

				plan, err :=
					conversionWorkloads(
						test.format,
						test.target,
					)

				if err != nil {
					t.Fatalf(
						"create workload plan: %v",
						err,
					)
				}

				if !reflect.DeepEqual(
					plan.Workloads,
					test.workloads,
				) {
					t.Fatalf(
						"workloads mismatch: expected %v, got %v",
						test.workloads,
						plan.Workloads,
					)
				}
			},
		)
	}
}

func TestConversionWorkloadsRejectsUnknownConversion(
	t *testing.T,
) {
	t.Parallel()

	_, err :=
		conversionWorkloads(
			media.Format{
				ID: "unknown",

				Category: media.Category(
					"unknown",
				),
			},
			"unknown",
		)

	if err == nil {
		t.Fatal(
			"expected unknown conversion workload mapping to fail",
		)
	}
}
