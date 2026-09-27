import { getDownloadFilename } from "../../shared/download.ts";

export interface PDFSortResult {
  blob: Blob;
  filename: string;
}

export async function sortPDFPages(
  uploadID: string,
  pages: number[],
): Promise<PDFSortResult> {
  const response = await fetch("/pdf/sort", {
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
      message.trim() || "PDF-Seiten konnten nicht sortiert werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "sortiert.pdf",
    ),
  };
}
