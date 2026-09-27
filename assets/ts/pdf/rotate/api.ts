import { getDownloadFilename } from "../../shared/download.ts";

export interface PDFRotateResult {
  blob: Blob;
  filename: string;
}

export async function rotatePDFPages(
  uploadID: string,
  pages: number[],
  angle: number,
): Promise<PDFRotateResult> {
  const response = await fetch("/pdf/rotate-pages", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      upload_id: uploadID,
      pages,
      angle,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(
      message.trim() || "Die ausgewählten Seiten konnten nicht gedreht werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "gedreht.pdf",
    ),
  };
}
