import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export async function mergePDFUploads(ids: string[]): Promise<DownloadResult> {
  return requestDownload(
    "/pdf/merge",
    {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        ids,
      }),
    },
    "PDFs konnten nicht zusammengefügt werden.",
  );
}
