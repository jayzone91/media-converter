export interface PDFSecurityUpload {
  id: string;
  filename: string;
  size: number;
}

interface PDFSecurityUploadResponse {
  id: string;
  filename: string;
  size: number;
}

export async function uploadPDFSecurityFile(
  file: File,
): Promise<PDFSecurityUpload> {
  const body = new FormData();

  body.append("file", file, file.name);

  const response = await fetch("/pdf/uploads/raw", {
    method: "POST",
    body,
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(message.trim() || "PDF konnte nicht hochgeladen werden.");
  }

  const data = (await response.json()) as unknown;

  if (!isPDFSecurityUploadResponse(data)) {
    throw new Error("Der Server hat eine ungültige Upload-Antwort geliefert.");
  }

  return {
    id: data.id,

    filename: data.filename,

    size: data.size,
  };
}

function isPDFSecurityUploadResponse(
  value: unknown,
): value is PDFSecurityUploadResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;

  return (
    typeof candidate.id === "string" &&
    typeof candidate.filename === "string" &&
    typeof candidate.size === "number" &&
    Number.isFinite(candidate.size) &&
    candidate.size > 0
  );
}
