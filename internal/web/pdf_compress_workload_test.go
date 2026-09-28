package web

import (
	"testing"
)

func TestPDFCompressionWorkloadPlan(
	t *testing.T,
) {
	t.Parallel()

	tests :=
		[]struct {
			mode string

			expected workloadType
		}{
			{
				mode: "lossless",

				expected: workloadQPDF,
			},
			{
				mode: "balanced",

				expected: workloadGhostscript,
			},
			{
				mode: "strong",

				expected: workloadGhostscript,
			},
		}

	for _, test := range tests {
		test :=
			test

		t.Run(
			test.mode,
			func(
				t *testing.T,
			) {
				t.Parallel()

				plan, err :=
					pdfCompressionWorkloadPlan(
						test.mode,
					)

				if err != nil {
					t.Fatalf(
						"create workload plan: %v",
						err,
					)
				}

				if len(
					plan.Workloads,
				) != 1 {
					t.Fatalf(
						"expected one workload, got %d",
						len(
							plan.Workloads,
						),
					)
				}

				if plan.Workloads[0] !=
					test.expected {
					t.Fatalf(
						"expected workload %q, got %q",
						test.expected,
						plan.Workloads[0],
					)
				}
			},
		)
	}
}

func TestPDFCompressionWorkloadPlanRejectsUnknownMode(
	t *testing.T,
) {
	t.Parallel()

	_, err :=
		pdfCompressionWorkloadPlan(
			"unknown",
		)

	if err == nil {
		t.Fatal(
			"expected unknown compression mode to fail",
		)
	}
}
