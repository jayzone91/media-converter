import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export interface PDFPageRotation {
  page: number;
  angle: 90 | 180 | 270;
}

export type PDFRotateResult = DownloadResult;

export async function rotatePDFPages(
  uploadID: string,
  rotations: PDFPageRotation[],
): Promise<PDFRotateResult> {
  return requestDownload(
    "/pdf/rotate-pages",
    {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        upload_id: uploadID,

        rotations,
      }),
    },
    "Die Seiten konnten nicht gedreht werden.",
  );
}
