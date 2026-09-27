import { getDownloadFilename } from "../../shared/download.ts";

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

export interface PDFCompressionResult {
  blob: Blob;
  filename: string;
  originalSize: number;
  resultSize: number;
  unchanged: boolean;
}

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
    const message = await response.text();

    throw new Error(
      message.trim() ||
        "Die mögliche Kompression konnte nicht berechnet werden.",
    );
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
  const response = await fetch("/pdf/compress", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      upload_id: uploadID,
      mode,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(
      message.trim() || "Die PDF konnte nicht komprimiert werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "komprimiert.pdf",
    ),

    originalSize: parseSizeHeader(response.headers.get("X-Original-Size")),

    resultSize: parseSizeHeader(response.headers.get("X-Result-Size")),

    unchanged: response.headers.get("X-Compression-Unchanged") === "true",
  };
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

function parseSizeHeader(value: string | null): number {
  if (!value) {
    return 0;
  }

  const size = Number.parseInt(value, 10);

  return Number.isFinite(size) ? size : 0;
}
