import { readAPIError } from "../../shared/api-error.ts";
import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export type PDFCompressionMode = "lossless" | "balanced" | "strong";

export interface PDFCompressionAnalysis {
  originalSize: number;
  resultSize: number;
  savingsBytes: number;
  savingsPercent: number;
  unchanged: boolean;
}

interface PDFCompressionAnalysisResponse {
  original_size: number;
  result_size: number;
  savings_bytes: number;
  savings_percent: number;
  unchanged: boolean;
}

export type PDFCompressionResult = DownloadResult;

export async function analyzePDFCompression(
  uploadID: string,
  mode: PDFCompressionMode,
  signal?: AbortSignal,
): Promise<PDFCompressionAnalysis> {
  const response = await fetch("/pdf/compress/analyze", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      upload_id: uploadID,
      mode,
    }),

    signal: signal ?? null,
  });

  if (!response.ok) {
    const error = await readAPIError(
      response,
      "Die mögliche Kompression konnte nicht berechnet werden.",
    );

    throw new Error(error.message);
  }

  const data = (await response.json()) as unknown;

  if (!isCompressionAnalysisResponse(data)) {
    throw new Error(
      "Der Server hat eine ungültige Kompressionsanalyse geliefert.",
    );
  }

  return {
    originalSize: data.original_size,
    resultSize: data.result_size,
    savingsBytes: data.savings_bytes,
    savingsPercent: data.savings_percent,
    unchanged: data.unchanged,
  };
}

export async function compressPDF(
  uploadID: string,
  mode: PDFCompressionMode,
): Promise<PDFCompressionResult> {
  return requestDownload(
    "/pdf/compress",
    {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        upload_id: uploadID,
        mode,
      }),
    },
    "Die PDF konnte nicht komprimiert werden.",
  );
}

function isCompressionAnalysisResponse(
  value: unknown,
): value is PDFCompressionAnalysisResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;

  return (
    isNonNegativeNumber(candidate.original_size) &&
    isNonNegativeNumber(candidate.result_size) &&
    isNonNegativeNumber(candidate.savings_bytes) &&
    isNonNegativeNumber(candidate.savings_percent) &&
    typeof candidate.unchanged === "boolean"
  );
}

function isNonNegativeNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= 0;
}
