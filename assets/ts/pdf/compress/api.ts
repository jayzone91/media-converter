import { getDownloadFilename } from "../../shared/download.ts";

export type PDFCompressionMode = "lossless" | "balanced" | "strong";

export interface PDFCompressionResult {
  blob: Blob;
  filename: string;
  originalSize: number;
  resultSize: number;
  unchanged: boolean;
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

function parseSizeHeader(value: string | null): number {
  if (!value) {
    return 0;
  }

  const size = Number.parseInt(value, 10);

  return Number.isFinite(size) ? size : 0;
}
