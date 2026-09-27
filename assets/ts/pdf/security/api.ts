import { requestDownload, type DownloadResult } from "../../shared/download.ts";

export type PDFSecurityResult = DownloadResult;

export async function encryptPDF(
  uploadID: string,
  password: string,
): Promise<PDFSecurityResult> {
  return requestPDFSecurityOperation("/pdf/encrypt", uploadID, password);
}

export async function decryptPDF(
  uploadID: string,
  password: string,
): Promise<PDFSecurityResult> {
  return requestPDFSecurityOperation("/pdf/decrypt", uploadID, password);
}

async function requestPDFSecurityOperation(
  url: string,
  uploadID: string,
  password: string,
): Promise<PDFSecurityResult> {
  return requestDownload(
    url,
    {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        upload_id: uploadID,

        password,
      }),
    },
    "Die PDF konnte nicht verarbeitet werden.",
  );
}
