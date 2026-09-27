import { getDownloadFilename } from "../../shared/download.ts";

export interface PDFExtractResult {
  blob: Blob;
  filename: string;
}

export async function extractPDFPages(
  uploadID: string,
  pages: number[],
): Promise<PDFExtractResult> {
  const response = await fetch("/pdf/extract-pages", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      upload_id: uploadID,
      pages,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(
      message.trim() ||
        "Die ausgewählten Seiten konnten nicht extrahiert werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "extrahiert.pdf",
    ),
  };
}
