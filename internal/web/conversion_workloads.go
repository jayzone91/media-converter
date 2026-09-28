package web

import (
	"context"
	"slices"

	"github.com/jayzone91/media-converter/internal/media"
)

type conversionWorkloadPlan struct {
	Workloads []workloadType

	UseLegacySlot bool
}

func conversionWorkloads(
	format media.Format,
	target string,
) conversionWorkloadPlan {
	switch format.Category {
	case media.CategoryImage:
		if format.ID == "gif" &&
			(target == "mp4" ||
				target == "webm") {
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadFFmpeg,
				},
			}
		}

		return conversionWorkloadPlan{
			Workloads: []workloadType{
				workloadImageMagick,
			},
		}

	case media.CategoryAudio,
		media.CategoryVideo:
		return conversionWorkloadPlan{
			Workloads: []workloadType{
				workloadFFmpeg,
			},
		}

	case media.CategoryDocument:
		if format.ID == "txt" &&
			target == "html" {
			return conversionWorkloadPlan{}
		}

		return conversionWorkloadPlan{
			Workloads: []workloadType{
				workloadLibreOffice,
			},
		}

	case media.CategoryMarkdown:
		switch target {
		case "html":
			return conversionWorkloadPlan{}

		case "pdf":
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadChromium,
				},
			}

		case "png",
			"jpeg",
			"webp":
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadChromium,
					workloadImageMagick,
				},
			}
		}

	case media.CategoryPDF:
		switch target {
		case "docx":
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadLibreOffice,
				},
			}

		case "png",
			"jpeg":
			/*
				Dieser Pfad arbeitet aktuell vor allem mit
				Poppler und ggf. weiteren PDF-Werkzeugen.

				Dafür existiert noch kein eigener Workload-Pool.
				Bis dieser Pfad separat klassifiziert wird,
				bleibt er durch den bisherigen Slot begrenzt.
			*/
			return conversionWorkloadPlan{
				UseLegacySlot: true,
			}
		}
	}

	return conversionWorkloadPlan{
		UseLegacySlot: true,
	}
}

func (s *Server) acquireConversionWorkloads(
	ctx context.Context,
	plan conversionWorkloadPlan,
) (
	func(),
	error,
) {
	if plan.UseLegacySlot {
		if err :=
			s.acquireConversionSlot(
				ctx,
			); err != nil {
			return nil, err
		}

		return func() {
			s.releaseConversionSlot()
		}, nil
	}

	if len(plan.Workloads) == 0 {
		return func() {}, nil
	}

	workloads :=
		append(
			[]workloadType(nil),
			plan.Workloads...,
		)

	slices.Sort(
		workloads,
	)

	acquired :=
		make(
			[]workloadType,
			0,
			len(workloads),
		)

	for _, workload := range workloads {
		if slices.Contains(
			acquired,
			workload,
		) {
			continue
		}

		if err :=
			s.acquireWorkload(
				ctx,
				workload,
			); err != nil {
			releaseConversionWorkloads(
				s,
				acquired,
			)

			return nil, err
		}

		acquired = append(
			acquired,
			workload,
		)
	}

	return func() {
		releaseConversionWorkloads(
			s,
			acquired,
		)
	}, nil
}

func releaseConversionWorkloads(
	server *Server,
	workloads []workloadType,
) {
	for index :=
		len(workloads) - 1; index >= 0; index-- {
		server.releaseWorkload(
			workloads[index],
		)
	}
}
