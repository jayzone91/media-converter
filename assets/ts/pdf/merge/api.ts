import { getDownloadFilename } from "../../shared/download.ts";
import type { MergeResult, PDFUploadResponse } from "./types.ts";

export async function uploadPDF(file: File): Promise<PDFUploadResponse> {
  const body = new FormData();

  body.append("file", file, file.name);

  const response = await fetch("/pdf/uploads", {
    method: "POST",
    body,
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(message.trim() || "PDF konnte nicht hochgeladen werden.");
  }

  const data = (await response.json()) as unknown;

  if (!isPDFUploadResponse(data)) {
    throw new Error("Der Server hat eine ungültige Upload-Antwort geliefert.");
  }

  return data;
}

export async function deletePDFUpload(id: string): Promise<void> {
  try {
    const response = await fetch(`/pdf/uploads/${encodeURIComponent(id)}`, {
      method: "DELETE",
    });

    if (!response.ok && response.status !== 404) {
      console.warn(`PDF upload ${id} could not be deleted.`);
    }
  } catch (error: unknown) {
    console.warn("PDF upload cleanup failed:", error);
  }
}

export async function mergePDFUploads(ids: string[]): Promise<MergeResult> {
  const response = await fetch("/pdf/merge", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      ids,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(
      message.trim() || "PDFs konnten nicht zusammengefügt werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "zusammengefuegt.pdf",
    ),
  };
}

function isPDFUploadResponse(value: unknown): value is PDFUploadResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;

  return (
    typeof candidate.id === "string" &&
    typeof candidate.filename === "string" &&
    typeof candidate.size === "number" &&
    Number.isFinite(candidate.size) &&
    candidate.size >= 0 &&
    typeof candidate.page_count === "number" &&
    Number.isInteger(candidate.page_count) &&
    candidate.page_count > 0 &&
    Array.isArray(candidate.previews) &&
    candidate.previews.length === candidate.page_count &&
    candidate.previews.every((preview) => typeof preview === "string")
  );
}
