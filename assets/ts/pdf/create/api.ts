import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export type PDFCreatePaperSize = "a4" | "letter";

export interface PDFCreateOptions {
  title: string;
  body: string;
  paperSize: PDFCreatePaperSize;
  landscape: boolean;
  fontSize: number;
  includeDate: boolean;
  markdown: boolean;
}

export type PDFCreateResult = DownloadResult;

export async function createPDFDocument(
  options: PDFCreateOptions,
): Promise<PDFCreateResult> {
  return requestDownload(
    "/pdf/create",
    {
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
        markdown: options.markdown,
      }),
    },
    "Das PDF-Dokument konnte nicht erstellt werden.",
  );
}
