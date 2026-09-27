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
