import { getDownloadFilename } from "../../shared/download.ts";

export type PDFCreatePaperSize = "a4" | "letter";

export interface PDFCreateOptions {
  title: string;
  body: string;
  paperSize: PDFCreatePaperSize;
  landscape: boolean;
  fontSize: number;
  includeDate: boolean;
}

export interface PDFCreateResult {
  blob: Blob;
  filename: string;
}

export async function createPDFDocument(
  options: PDFCreateOptions,
): Promise<PDFCreateResult> {
  const response = await fetch("/pdf/create", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      title: options.title,
      body: options.body,
      paper_size: options.paperSize,
      landscape: options.landscape,
      font_size: options.fontSize,
      include_date: options.includeDate,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(
      message.trim() || "Das PDF-Dokument konnte nicht erstellt werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "dokument.pdf",
    ),
  };
}
