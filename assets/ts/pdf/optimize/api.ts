import { readAPIError } from "../../shared/api-error.ts";
import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export interface PDFOptimizationAnalysis {
  originalSize: number;
  resultSize: number;
  deltaBytes: number;
  deltaPercent: number;
  linearized: boolean;
  unchanged: boolean;
}

interface PDFOptimizationAnalysisResponse {
  original_size: number;
  result_size: number;
  delta_bytes: number;
  delta_percent: number;
  linearized: boolean;
  unchanged: boolean;
}

export type PDFOptimizationResult = DownloadResult;

export async function analyzePDFOptimization(
  uploadID: string,
  linearize: boolean,
  signal?: AbortSignal,
): Promise<PDFOptimizationAnalysis> {
  const response = await fetch("/pdf/optimize/analyze", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      upload_id: uploadID,

      linearize,
    }),

    signal: signal ?? null,
  });

  if (!response.ok) {
    const error = await readAPIError(
      response,
      "Die PDF-Optimierung konnte nicht berechnet werden.",
    );

    throw new Error(error.message);
  }

  const data = (await response.json()) as unknown;

  if (!isAnalysisResponse(data)) {
    throw new Error(
      "Der Server hat eine ungültige Optimierungsanalyse geliefert.",
    );
  }

  return {
    originalSize: data.original_size,

    resultSize: data.result_size,

    deltaBytes: data.delta_bytes,

    deltaPercent: data.delta_percent,

    linearized: data.linearized,

    unchanged: data.unchanged,
  };
}

export async function optimizePDF(
  uploadID: string,
  linearize: boolean,
): Promise<PDFOptimizationResult> {
  return requestDownload(
    "/pdf/optimize",
    {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        upload_id: uploadID,

        linearize,
      }),
    },
    "Die PDF konnte nicht optimiert werden.",
  );
}

function isAnalysisResponse(
  value: unknown,
): value is PDFOptimizationAnalysisResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;

  return (
    isNumber(candidate.original_size) &&
    isNumber(candidate.result_size) &&
    isNumber(candidate.delta_bytes) &&
    isNumber(candidate.delta_percent) &&
    typeof candidate.linearized === "boolean" &&
    typeof candidate.unchanged === "boolean"
  );
}

function isNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}
