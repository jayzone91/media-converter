import { downloadBlob } from "../../shared/download.ts";

import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import { type PDFPageRotation, rotatePDFPages } from "./api.ts";

import { createPDFRotationGrid, type PDFRotationGrid } from "./grid.ts";

import type { RotatePage } from "./types.ts";

const MAX_FILE_SIZE = 512 * 1024 * 1024;

let root: HTMLElement | null = null;

let upload: PDFUpload | null = null;

let grid: PDFRotationGrid | null = null;

let rotatedPages: RotatePage[] = [];

let uploading = false;

let processing = false;

export function setupPDFRotate(workspace: HTMLElement): void {
  grid?.destroy();

  root = workspace;

  upload = null;

  rotatedPages = [];

  uploading = false;
  processing = false;

  grid = createPDFRotationGrid({
    root: workspace,

    onChange: handleRotationChange,
  });

  const input = getElement<HTMLInputElement>("#pdf-rotate-file");

  const dropZone = getElement<HTMLElement>("#pdf-rotate-drop-zone");

  const resetFile = getElement<HTMLButtonElement>("#pdf-rotate-reset-file");

  const resetRotations = getElement<HTMLButtonElement>(
    "#pdf-rotate-reset-rotations",
  );

  const submit = getElement<HTMLButtonElement>("#pdf-rotate-submit");

  if (!input || !dropZone || !resetFile || !resetRotations || !submit) {
    grid.destroy();

    grid = null;

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

  resetFile.addEventListener("click", () => {
    void resetUpload();
  });

  resetRotations.addEventListener("click", () => {
    grid?.reset();
  });

  submit.addEventListener("click", () => {
    void createPDF();
  });

  render();
}

export async function destroyPDFRotate(): Promise<void> {
  const uploadID = upload?.id;

  grid?.destroy();

  grid = null;
  root = null;
  upload = null;

  rotatedPages = [];

  uploading = false;
  processing = false;

  if (uploadID) {
    await deletePDFUpload(uploadID);
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
      showError("Es kann nur eine PDF gleichzeitig bearbeitet werden.");

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

  rotatedPages = [];

  grid?.setPages([]);

  uploading = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-rotate-upload-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    upload = await uploadPDF(file);

    grid?.setPages(upload.previews);

    render();
  } catch (error: unknown) {
    upload = null;

    rotatedPages = [];

    grid?.setPages([]);

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

  rotatedPages = [];

  grid?.setPages([]);

  render();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

function handleRotationChange(pages: RotatePage[]): void {
  rotatedPages = pages;

  updateRotationText();

  updateControls();
}

async function createPDF(): Promise<void> {
  if (!upload || rotatedPages.length === 0 || uploading || processing) {
    return;
  }

  clearError();

  processing = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-rotate-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const rotations: PDFPageRotation[] = rotatedPages.map((page) => ({
      page: page.page,

      angle: page.rotation as 90 | 180 | 270,
    }));

    const result = await rotatePDFPages(upload.id, rotations);

    downloadBlob(result.blob, result.filename);

    upload = null;

    rotatedPages = [];

    grid?.setPages([]);

    render();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Die Seiten konnten nicht gedreht werden.",
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
  const editor = getElement<HTMLElement>("#pdf-rotate-editor");

  const dropZone = getElement<HTMLElement>("#pdf-rotate-drop-zone");

  const filename = getElement<HTMLElement>("#pdf-rotate-filename");

  const meta = getElement<HTMLElement>("#pdf-rotate-meta");

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

  updateRotationText();

  updateControls();
}

function updateRotationText(): void {
  const element = getElement<HTMLElement>("#pdf-rotate-changed-count");

  if (!element) {
    return;
  }

  if (rotatedPages.length === 0) {
    element.textContent = "Keine Seiten geändert";

    return;
  }

  if (rotatedPages.length === 1) {
    element.textContent = "1 Seite geändert";

    return;
  }

  element.textContent = `${rotatedPages.length} Seiten geändert`;
}

function updateControls(): void {
  const input = getElement<HTMLInputElement>("#pdf-rotate-file");

  const resetFile = getElement<HTMLButtonElement>("#pdf-rotate-reset-file");

  const resetRotations = getElement<HTMLButtonElement>(
    "#pdf-rotate-reset-rotations",
  );

  const submit = getElement<HTMLButtonElement>("#pdf-rotate-submit");

  const busy = uploading || processing;

  if (input) {
    input.disabled = busy;
  }

  if (resetFile) {
    resetFile.disabled = !upload || busy;
  }

  if (resetRotations) {
    resetRotations.disabled = rotatedPages.length === 0 || busy;
  }

  if (submit) {
    submit.disabled = !upload || rotatedPages.length === 0 || busy;
  }
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-rotate-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-rotate-error");

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
