import { getDownloadFilename } from "../../shared/download.ts";

export interface PDFPageRotation {
  page: number;
  angle: 90 | 180 | 270;
}

export interface PDFRotateResult {
  blob: Blob;
  filename: string;
}

export async function rotatePDFPages(
  uploadID: string,
  rotations: PDFPageRotation[],
): Promise<PDFRotateResult> {
  const response = await fetch("/pdf/rotate-pages", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      upload_id: uploadID,
      rotations,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(
      message.trim() || "Die Seiten konnten nicht gedreht werden.",
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
