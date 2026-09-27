import { downloadBlob } from "../../shared/download.ts";

import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import { compressPDF, type PDFCompressionMode } from "./api.ts";

const MAX_FILE_SIZE = 512 * 1024 * 1024;

let root: HTMLElement | null = null;

let upload: PDFUpload | null = null;

let mode: PDFCompressionMode = "balanced";

let uploading = false;

let processing = false;

export function setupPDFCompress(workspace: HTMLElement): void {
  root = workspace;

  upload = null;

  mode = "balanced";

  uploading = false;

  processing = false;

  const input = getElement<HTMLInputElement>("#pdf-compress-file");

  const dropZone = getElement<HTMLElement>("#pdf-compress-drop-zone");

  const reset = getElement<HTMLButtonElement>("#pdf-compress-reset-file");

  const submit = getElement<HTMLButtonElement>("#pdf-compress-submit");

  if (!input || !dropZone || !reset || !submit) {
    return;
  }

  input.addEventListener("change", () => {
    const file = input.files?.[0];

    input.value = "";

    if (file) {
      void selectFile(file);
    }
  });

  setupDropZone(dropZone);

  setupModeButtons();

  reset.addEventListener("click", () => {
    void resetUpload();
  });

  submit.addEventListener("click", () => {
    void createCompressedPDF();
  });

  render();
}

export async function destroyPDFCompress(): Promise<void> {
  const uploadID = upload?.id;

  root = null;

  upload = null;

  mode = "balanced";

  uploading = false;

  processing = false;

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

function setupModeButtons(): void {
  const buttons = root?.querySelectorAll<HTMLButtonElement>(
    "[data-compression-mode]",
  );

  if (!buttons) {
    return;
  }

  for (const button of buttons) {
    button.addEventListener("click", () => {
      const candidate = button.dataset.compressionMode;

      if (!isCompressionMode(candidate)) {
        return;
      }

      mode = candidate;

      for (const other of buttons) {
        const selected = other === button;

        other.classList.toggle("selected", selected);

        other.setAttribute("aria-pressed", selected ? "true" : "false");
      }

      updateNotice();
    });
  }
}

function setupDropZone(dropZone: HTMLElement): void {
  const events = ["dragenter", "dragover", "dragleave", "drop"] as const;

  for (const eventName of events) {
    dropZone.addEventListener(eventName, (event) => {
      event.preventDefault();
      event.stopPropagation();
    });
  }

  for (const eventName of ["dragenter", "dragover"] as const) {
    dropZone.addEventListener(eventName, () => {
      dropZone.classList.add("drag-over");
    });
  }

  for (const eventName of ["dragleave", "drop"] as const) {
    dropZone.addEventListener(eventName, () => {
      dropZone.classList.remove("drag-over");
    });
  }

  dropZone.addEventListener("drop", (event: DragEvent) => {
    const files = event.dataTransfer?.files;

    if (!files?.length) {
      return;
    }

    if (files.length > 1) {
      showError("Es kann nur eine PDF gleichzeitig komprimiert werden.");

      return;
    }

    const file = files[0];

    if (file) {
      void selectFile(file);
    }
  });
}

async function selectFile(file: File): Promise<void> {
  if (uploading || processing) {
    return;
  }

  clearError();

  if (!isPDFFile(file)) {
    showError("Es können nur PDF-Dateien hochgeladen werden.");

    return;
  }

  if (file.size <= 0) {
    showError("Die PDF-Datei ist leer.");

    return;
  }

  if (file.size > MAX_FILE_SIZE) {
    showError("Die PDF-Datei ist größer als 512 MiB.");

    return;
  }

  if (upload) {
    await deletePDFUpload(upload.id);

    upload = null;
  }

  uploading = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-compress-upload-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    upload = await uploadPDF(file);

    render();
  } catch (error: unknown) {
    upload = null;

    showError(
      error instanceof Error
        ? `${file.name}: ${error.message}`
        : `${file.name}: Upload fehlgeschlagen.`,
    );
  } finally {
    uploading = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

async function resetUpload(): Promise<void> {
  if (uploading || processing) {
    return;
  }

  const uploadID = upload?.id;

  upload = null;

  render();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

async function createCompressedPDF(): Promise<void> {
  if (!upload || uploading || processing) {
    return;
  }

  clearError();

  processing = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-compress-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await compressPDF(upload.id, mode);

    downloadBlob(result.blob, result.filename);

    if (result.unchanged) {
      showError(
        "Die gewählte Kompression konnte die Datei nicht weiter verkleinern. Das Original wurde ausgegeben.",
      );
    }

    upload = null;

    render();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Die PDF konnte nicht komprimiert werden.",
    );
  } finally {
    processing = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

function render(): void {
  const editor = getElement<HTMLElement>("#pdf-compress-editor");

  const dropZone = getElement<HTMLElement>("#pdf-compress-drop-zone");

  const filename = getElement<HTMLElement>("#pdf-compress-filename");

  const meta = getElement<HTMLElement>("#pdf-compress-meta");

  if (editor) {
    editor.hidden = upload === null;
  }

  if (dropZone) {
    dropZone.hidden = upload !== null;
  }

  if (upload && filename) {
    filename.textContent = upload.filename;
  }

  if (upload && meta) {
    meta.textContent = `${upload.pageCount} Seiten · ${formatBytes(upload.size)}`;
  }

  updateNotice();

  updateControls();
}

function updateNotice(): void {
  const notice = getElement<HTMLElement>("#pdf-compress-notice");

  if (!notice) {
    return;
  }

  const title = notice.querySelector<HTMLElement>("strong");

  const text = notice.querySelector<HTMLElement>("span");

  if (!title || !text) {
    return;
  }

  switch (mode) {
    case "lossless":
      title.textContent = "Verlustfrei";

      text.textContent =
        "Keine Änderung an Bildauflösung oder Bildqualität. Die mögliche Ersparnis kann gering sein.";

      break;

    case "balanced":
      title.textContent = "Ausgewogen";

      text.textContent =
        "Geeignet für Bildschirmdarstellung, E-Mail und typische Dokumente.";

      break;

    case "strong":
      title.textContent = "Stark";

      text.textContent =
        "Für möglichst kleine Dateien. Bilder können sichtbar an Detail verlieren.";

      break;
  }
}

function updateControls(): void {
  const input = getElement<HTMLInputElement>("#pdf-compress-file");

  const reset = getElement<HTMLButtonElement>("#pdf-compress-reset-file");

  const submit = getElement<HTMLButtonElement>("#pdf-compress-submit");

  const buttons = root?.querySelectorAll<HTMLButtonElement>(
    "[data-compression-mode]",
  );

  const busy = uploading || processing;

  if (input) {
    input.disabled = busy;
  }

  if (reset) {
    reset.disabled = !upload || busy;
  }

  if (submit) {
    submit.disabled = !upload || busy;
  }

  buttons?.forEach((button) => {
    button.disabled = busy;
  });
}

function isCompressionMode(
  value: string | undefined,
): value is PDFCompressionMode {
  return value === "lossless" || value === "balanced" || value === "strong";
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-compress-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-compress-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}

function formatBytes(bytes: number): string {
  const units = ["B", "KiB", "MiB", "GiB"] as const;

  if (bytes <= 0) {
    return "0 B";
  }

  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );

  const value = bytes / 1024 ** index;

  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index] ?? "B"}`;
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
