import { getDownloadFilename } from "../../shared/download.ts";

export interface PDFDeleteResult {
  blob: Blob;

  filename: string;
}

export async function deletePDFPages(
  uploadID: string,
  pages: number[],
): Promise<PDFDeleteResult> {
  const response = await fetch("/pdf/delete-pages", {
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
        "Die ausgewählten Seiten konnten nicht entfernt werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "seiten-entfernt.pdf",
    ),
  };
}
