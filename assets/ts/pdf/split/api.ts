import { getDownloadFilename } from "../../shared/download.ts";

export interface PDFSplitResult {
  blob: Blob;
  filename: string;
}

export async function splitPDF(
  uploadID: string,
  splitAfter: number[],
): Promise<PDFSplitResult> {
  const response = await fetch("/pdf/split", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      upload_id: uploadID,

      split_after: splitAfter,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(message.trim() || "Die PDF konnte nicht getrennt werden.");
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "getrennte-pdfs.zip",
    ),
  };
}
