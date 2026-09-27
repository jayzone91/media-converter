import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export type PDFSortResult = DownloadResult;

export async function sortPDFPages(
  uploadID: string,
  pages: number[],
): Promise<PDFSortResult> {
  return requestDownload(
    "/pdf/sort",
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
    "PDF-Seiten konnten nicht sortiert werden.",
  );
}
