import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export type PDFExtractResult = DownloadResult;

export async function extractPDFPages(
  uploadID: string,
  pages: number[],
): Promise<PDFExtractResult> {
  return requestDownload(
    "/pdf/extract-pages",
    {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        upload_id: uploadID,

        pages,
      }),
    },
    "Die ausgewählten Seiten konnten nicht extrahiert werden.",
  );
}
