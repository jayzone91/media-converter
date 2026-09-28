package web

import (
	"context"
	"fmt"
	"slices"

	"github.com/jayzone91/media-converter/internal/media"
)

type conversionWorkloadPlan struct {
	Workloads []workloadType
}

func conversionWorkloads(
	format media.Format,
	target string,
) (
	conversionWorkloadPlan,
	error,
) {
	switch format.Category {
	case media.CategoryImage:
		if format.ID == "gif" &&
			(target == "mp4" ||
				target == "webm") {
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadFFmpeg,
				},
			}, nil
		}

		return conversionWorkloadPlan{
			Workloads: []workloadType{
				workloadImageMagick,
			},
		}, nil

	case media.CategoryAudio,
		media.CategoryVideo:
		return conversionWorkloadPlan{
			Workloads: []workloadType{
				workloadFFmpeg,
			},
		}, nil

	case media.CategoryDocument:
		if format.ID == "txt" &&
			target == "html" {
			return conversionWorkloadPlan{},
				nil
		}

		return conversionWorkloadPlan{
			Workloads: []workloadType{
				workloadLibreOffice,
			},
		}, nil

	case media.CategoryMarkdown:
		switch target {
		case "html":
			return conversionWorkloadPlan{},
				nil

		case "pdf":
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadChromium,
				},
			}, nil

		case "png",
			"jpeg",
			"webp":
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadChromium,
					workloadImageMagick,
				},
			}, nil
		}

	case media.CategoryPDF:
		switch target {
		case "docx":
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadLibreOffice,
					workloadPoppler,
				},
			}, nil

		case "png",
			"jpeg":
			return conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadPoppler,
				},
			}, nil
		}
	}

	return conversionWorkloadPlan{},
		fmt.Errorf(
			"no workload mapping for conversion %s -> %s",
			format.ID,
			target,
		)
}

func (s *Server) acquireConversionWorkloads(
	ctx context.Context,
	plan conversionWorkloadPlan,
) (
	func(),
	error,
) {
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
