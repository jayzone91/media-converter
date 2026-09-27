import { downloadBlob } from "../../shared/download.ts";

import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import { createPDFSplitGrid, type PDFSplitGrid } from "./grid.ts";

import { splitPDF } from "./api.ts";

import type { PDFSplitSegment } from "./types.ts";

const MAX_FILE_SIZE = 512 * 1024 * 1024;

let root: HTMLElement | null = null;

let upload: PDFUpload | null = null;

let grid: PDFSplitGrid | null = null;

let splitPoints: number[] = [];

let uploading = false;

let processing = false;

export function setupPDFSplit(workspace: HTMLElement): void {
  grid?.destroy();

  root = workspace;

  upload = null;

  splitPoints = [];

  uploading = false;

  processing = false;

  grid = createPDFSplitGrid({
    root: workspace,

    onChange: handleSplitChange,

    onLimitReached: () => {
      showError("Es können maximal 200 Teildokumente erzeugt werden.");
    },
  });

  const input = getElement<HTMLInputElement>("#pdf-split-file");

  const dropZone = getElement<HTMLElement>("#pdf-split-drop-zone");

  const resetFile = getElement<HTMLButtonElement>("#pdf-split-reset-file");

  const clear = getElement<HTMLButtonElement>("#pdf-split-clear");

  const submit = getElement<HTMLButtonElement>("#pdf-split-submit");

  if (!input || !dropZone || !resetFile || !clear || !submit) {
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

  clear.addEventListener("click", () => {
    grid?.clear();
  });

  submit.addEventListener("click", () => {
    void createSplit();
  });

  render();
}

export async function destroyPDFSplit(): Promise<void> {
  const uploadID = upload?.id;

  grid?.destroy();

  grid = null;

  root = null;

  upload = null;

  splitPoints = [];

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

  splitPoints = [];

  grid?.setPages([]);

  uploading = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-split-upload-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    upload = await uploadPDF(file);

    if (upload.pageCount < 2) {
      const uploadID = upload.id;

      upload = null;

      await deletePDFUpload(uploadID);

      throw new Error("Die PDF muss mindestens zwei Seiten enthalten.");
    }

    grid?.setPages(upload.previews);

    render();
  } catch (error: unknown) {
    upload = null;

    splitPoints = [];

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

  splitPoints = [];

  grid?.setPages([]);

  render();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

function handleSplitChange(points: number[]): void {
  splitPoints = points;

  clearError();

  updateSummary();

  updateControls();
}

async function createSplit(): Promise<void> {
  if (!upload || splitPoints.length === 0 || uploading || processing) {
    return;
  }

  clearError();

  processing = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-split-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await splitPDF(upload.id, splitPoints);

    downloadBlob(result.blob, result.filename);

    upload = null;

    splitPoints = [];

    grid?.setPages([]);

    render();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Die PDF konnte nicht getrennt werden.",
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
  const editor = getElement<HTMLElement>("#pdf-split-editor");

  const dropZone = getElement<HTMLElement>("#pdf-split-drop-zone");

  const filename = getElement<HTMLElement>("#pdf-split-filename");

  const meta = getElement<HTMLElement>("#pdf-split-meta");

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

  updateSummary();

  updateControls();
}

function updateSummary(): void {
  const count = getElement<HTMLElement>("#pdf-split-document-count");

  const ranges = getElement<HTMLElement>("#pdf-split-ranges");

  if (!count || !ranges) {
    return;
  }

  if (!upload) {
    count.textContent = "1 Dokument";

    ranges.textContent = "Noch keine Trennpunkte gesetzt";

    return;
  }

  const segments = buildSegments(upload.pageCount, splitPoints);

  count.textContent =
    segments.length === 1 ? "1 Dokument" : `${segments.length} Dokumente`;

  if (splitPoints.length === 0) {
    ranges.textContent = "Noch keine Trennpunkte gesetzt";

    return;
  }

  ranges.textContent = formatSegments(segments);
}

function buildSegments(pageCount: number, points: number[]): PDFSplitSegment[] {
  const sorted = [...points].sort((left, right) => left - right);

  const segments: PDFSplitSegment[] = [];

  let start = 1;

  for (const end of sorted) {
    segments.push({
      start,
      end,
    });

    start = end + 1;
  }

  segments.push({
    start,
    end: pageCount,
  });

  return segments;
}

function formatSegments(segments: PDFSplitSegment[]): string {
  const visible = segments.slice(0, 8);

  const labels = visible.map((segment) =>
    segment.start === segment.end
      ? `Seite ${segment.start}`
      : `Seiten ${segment.start}–${segment.end}`,
  );

  const remaining = segments.length - visible.length;

  if (remaining > 0) {
    labels.push(`+ ${remaining} weitere`);
  }

  return labels.join(" · ");
}

function updateControls(): void {
  const input = getElement<HTMLInputElement>("#pdf-split-file");

  const resetFile = getElement<HTMLButtonElement>("#pdf-split-reset-file");

  const clear = getElement<HTMLButtonElement>("#pdf-split-clear");

  const submit = getElement<HTMLButtonElement>("#pdf-split-submit");

  const busy = uploading || processing;

  if (input) {
    input.disabled = busy;
  }

  if (resetFile) {
    resetFile.disabled = !upload || busy;
  }

  if (clear) {
    clear.disabled = splitPoints.length === 0 || busy;
  }

  if (submit) {
    submit.disabled = !upload || splitPoints.length === 0 || busy;
  }
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-split-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-split-error");

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
