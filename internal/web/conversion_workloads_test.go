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
			legacy    bool
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
				name: "pdf to docx uses LibreOffice pool",

				format: media.Format{
					ID: "pdf",

					Category: media.CategoryPDF,
				},

				target: "docx",

				workloads: []workloadType{
					workloadLibreOffice,
				},
			},
			{
				name: "pdf to image remains legacy",

				format: media.Format{
					ID: "pdf",

					Category: media.CategoryPDF,
				},

				target: "png",

				legacy: true,
			},
			{
				name: "unknown conversion remains legacy",

				format: media.Format{
					ID: "unknown",

					Category: media.Category(
						"unknown",
					),
				},

				target: "unknown",

				legacy: true,
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

				plan :=
					conversionWorkloads(
						test.format,
						test.target,
					)

				if plan.UseLegacySlot !=
					test.legacy {
					t.Fatalf(
						"legacy mismatch: expected %v, got %v",
						test.legacy,
						plan.UseLegacySlot,
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
