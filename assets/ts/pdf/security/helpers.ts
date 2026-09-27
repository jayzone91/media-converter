export const MAX_PDF_FILE_SIZE = 512 * 1024 * 1024;

export function isPDFSecurityFile(file: File): boolean {
  return (
    file.type === "application/pdf" || file.name.toLowerCase().endsWith(".pdf")
  );
}

export function validatePDFSecurityFile(file: File): string | null {
  if (!isPDFSecurityFile(file)) {
    return "Es können nur PDF-Dateien hochgeladen werden.";
  }

  if (file.size <= 0) {
    return "Die PDF-Datei ist leer.";
  }

  if (file.size > MAX_PDF_FILE_SIZE) {
    return "Die PDF-Datei ist größer als 512 MiB.";
  }

  return null;
}

export function setupPDFSecurityDropZone(
  element: HTMLElement,
  onFile: (file: File) => void,
  onError: (message: string) => void,
): void {
  const events = ["dragenter", "dragover", "dragleave", "drop"] as const;

  for (const name of events) {
    element.addEventListener(name, (event) => {
      event.preventDefault();
      event.stopPropagation();
    });
  }

  for (const name of ["dragenter", "dragover"] as const) {
    element.addEventListener(name, () => {
      element.classList.add("drag-over");
    });
  }

  for (const name of ["dragleave", "drop"] as const) {
    element.addEventListener(name, () => {
      element.classList.remove("drag-over");
    });
  }

  element.addEventListener("drop", (event: DragEvent) => {
    const files = event.dataTransfer?.files;

    if (!files?.length) {
      return;
    }

    if (files.length > 1) {
      onError("Es kann nur eine PDF gleichzeitig verarbeitet werden.");

      return;
    }

    const file = files[0];

    if (file) {
      onFile(file);
    }
  });
}

export function formatPDFSecuritySize(bytes: number): string {
  const units = ["B", "KiB", "MiB", "GiB"] as const;

  if (bytes <= 0) {
    return "0 B";
  }

  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );

  const value = bytes / 1024 ** index;

  return `${value.toLocaleString("de-DE", {
    minimumFractionDigits: index === 0 ? 0 : 1,

    maximumFractionDigits: 1,
  })} ${units[index] ?? "B"}`;
}
