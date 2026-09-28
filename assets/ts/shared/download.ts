import { readAPIError } from "./api-error.ts";

export interface DownloadResponse {
  download_url: string;
  filename: string;
}

export interface DownloadResult {
  downloadURL: string;
  filename: string;
}

export function getDownloadFilename(
  contentDisposition: string | null,
  fallback: string,
): string {
  if (!contentDisposition) {
    return fallback;
  }

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);

  const encodedFilename = utf8Match?.[1];

  if (encodedFilename) {
    try {
      return decodeURIComponent(encodedFilename);
    } catch {
      return encodedFilename;
    }
  }

  const filenameMatch = contentDisposition.match(/filename="([^"]+)"/i);

  return filenameMatch?.[1] ?? fallback;
}

export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);

  const link = document.createElement("a");

  link.href = url;
  link.download = filename;

  document.body.appendChild(link);

  link.click();
  link.remove();

  window.setTimeout(() => {
    URL.revokeObjectURL(url);
  }, 1000);
}

export function downloadURL(url: string): void {
  const link = document.createElement("a");

  link.href = url;

  document.body.appendChild(link);

  link.click();
  link.remove();
}

export async function readDownloadResponse(
  response: Response,
  fallbackError: string,
): Promise<DownloadResult> {
  if (!response.ok) {
    const error = await readAPIError(response, fallbackError);

    throw new Error(error.message);
  }

  const data = (await response.json()) as unknown;

  if (!isDownloadResponse(data)) {
    throw new Error(
      "Der Server hat eine ungültige Download-Antwort geliefert.",
    );
  }

  return {
    downloadURL: data.download_url,
    filename: data.filename,
  };
}

export async function requestDownload(
  url: string,
  init: RequestInit,
  fallbackError: string,
): Promise<DownloadResult> {
  const response = await fetch(url, init);

  return readDownloadResponse(response, fallbackError);
}

function isDownloadResponse(value: unknown): value is DownloadResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;

  return (
    typeof candidate.download_url === "string" &&
    candidate.download_url.startsWith("/downloads/") &&
    typeof candidate.filename === "string" &&
    candidate.filename.length > 0
  );
}
