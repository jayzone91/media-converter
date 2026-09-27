import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export type PDFSplitResult = DownloadResult;

export async function splitPDF(
  uploadID: string,
  splitAfter: number[],
): Promise<PDFSplitResult> {
  return requestDownload(
    "/pdf/split",
    {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        upload_id: uploadID,

        split_after: splitAfter,
      }),
    },
    "Die PDF konnte nicht getrennt werden.",
  );
}
