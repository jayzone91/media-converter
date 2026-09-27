import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export type PDFDeleteResult = DownloadResult;

export async function deletePDFPages(
  uploadID: string,
  pages: number[],
): Promise<PDFDeleteResult> {
  return requestDownload(
    "/pdf/delete-pages",
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
    "Die ausgewählten Seiten konnten nicht entfernt werden.",
  );
}
