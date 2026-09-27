import { getDownloadFilename } from "../../shared/download.ts";

export interface PDFSecurityResult {
  blob: Blob;
  filename: string;
}

export async function encryptPDF(
  uploadID: string,
  password: string,
): Promise<PDFSecurityResult> {
  return requestPDFSecurityOperation(
    "/pdf/encrypt",
    uploadID,
    password,
    "verschluesselt.pdf",
  );
}

export async function decryptPDF(
  uploadID: string,
  password: string,
): Promise<PDFSecurityResult> {
  return requestPDFSecurityOperation(
    "/pdf/decrypt",
    uploadID,
    password,
    "ohne-passwort.pdf",
  );
}

async function requestPDFSecurityOperation(
  url: string,
  uploadID: string,
  password: string,
  fallbackFilename: string,
): Promise<PDFSecurityResult> {
  const response = await fetch(url, {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      upload_id: uploadID,

      password,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(
      message.trim() || "Die PDF konnte nicht verarbeitet werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      fallbackFilename,
    ),
  };
}
