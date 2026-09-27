import { downloadBlob } from "../../shared/download.ts";

import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import {
  analyzePDFCompression,
  compressPDF,
  type PDFCompressionAnalysis,
  type PDFCompressionMode,
} from "./api.ts";

const MAX_FILE_SIZE = 512 * 1024 * 1024;

let root: HTMLElement | null = null;

let upload: PDFUpload | null = null;

let mode: PDFCompressionMode = "balanced";

let analysis: PDFCompressionAnalysis | null = null;

const analysisCache = new Map<PDFCompressionMode, PDFCompressionAnalysis>();

let analysisController: AbortController | null = null;

let analysisRequestID = 0;

let uploading = false;

let analyzing = false;

let processing = false;

export function setupPDFCompress(workspace: HTMLElement): void {
  root = workspace;

  upload = null;

  mode = "balanced";

  analysis = null;

  analysisCache.clear();

  analysisController = null;

  analysisRequestID = 0;

  uploading = false;

  analyzing = false;

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
    void downloadCompressedPDF();
  });

  render();
}

export async function destroyPDFCompress(): Promise<void> {
  analysisController?.abort();

  analysisController = null;

  analysisRequestID++;

  const uploadID = upload?.id;

  root = null;

  upload = null;

  analysis = null;

  analysisCache.clear();

  mode = "balanced";

  uploading = false;

  analyzing = false;

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

      if (!isCompressionMode(candidate) || candidate === mode) {
        return;
      }

      mode = candidate;

      for (const other of buttons) {
        const selected = other === button;

        other.classList.toggle("selected", selected);

        other.setAttribute("aria-pressed", selected ? "true" : "false");
      }

      updateNotice();

      if (!upload) {
        return;
      }

      const cached = analysisCache.get(mode);

      if (cached) {
        analysisController?.abort();

        analysis = cached;

        analyzing = false;

        renderAnalysis();

        updateControls();

        return;
      }

      void analyzeCurrentMode();
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

  analysisController?.abort();

  analysis = null;

  analysisCache.clear();

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

  let uploaded = false;

  try {
    upload = await uploadPDF(file);

    uploaded = true;

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

  if (uploaded) {
    await analyzeCurrentMode();
  }
}

async function analyzeCurrentMode(): Promise<void> {
  if (!upload || uploading || processing) {
    return;
  }

  const cached = analysisCache.get(mode);

  if (cached) {
    analysis = cached;

    analyzing = false;

    renderAnalysis();

    updateControls();

    return;
  }

  analysisController?.abort();

  const controller = new AbortController();

  analysisController = controller;

  const requestID = ++analysisRequestID;

  const uploadID = upload.id;

  const requestedMode = mode;

  analysis = null;

  analyzing = true;

  clearError();

  renderAnalysis();

  updateControls();

  try {
    const result = await analyzePDFCompression(
      uploadID,
      requestedMode,
      controller.signal,
    );

    if (
      requestID !== analysisRequestID ||
      !upload ||
      upload.id !== uploadID ||
      mode !== requestedMode
    ) {
      return;
    }

    analysisCache.set(requestedMode, result);

    analysis = result;
  } catch (error: unknown) {
    if (error instanceof DOMException && error.name === "AbortError") {
      return;
    }

    if (requestID === analysisRequestID) {
      showError(
        error instanceof Error
          ? error.message
          : "Die mögliche Kompression konnte nicht berechnet werden.",
      );
    }
  } finally {
    if (requestID === analysisRequestID) {
      analyzing = false;

      analysisController = null;

      renderAnalysis();

      updateControls();
    }
  }
}

async function resetUpload(): Promise<void> {
  if (uploading || processing) {
    return;
  }

  analysisController?.abort();

  analysisController = null;

  analysisRequestID++;

  const uploadID = upload?.id;

  upload = null;

  analysis = null;

  analysisCache.clear();

  analyzing = false;

  render();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

async function downloadCompressedPDF(): Promise<void> {
  if (!upload || !analysis || uploading || analyzing || processing) {
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

    upload = null;

    analysis = null;

    analysisCache.clear();

    render();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Die PDF konnte nicht heruntergeladen werden.",
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

  renderAnalysis();

  updateControls();
}

function renderAnalysis(): void {
  const original = getElement<HTMLElement>("#pdf-compress-original-size");

  const result = getElement<HTMLElement>("#pdf-compress-result-size");

  const savings = getElement<HTMLElement>("#pdf-compress-savings-size");

  const percent = getElement<HTMLElement>("#pdf-compress-savings-percent");

  const status = getElement<HTMLElement>("#pdf-compress-analysis-status");

  if (!original || !result || !savings || !percent || !status) {
    return;
  }

  original.textContent = upload ? formatBytes(upload.size) : "—";

  if (analyzing) {
    result.textContent = "…";

    savings.textContent = "…";

    percent.textContent = "…";

    status.textContent = "Kompression wird berechnet …";

    return;
  }

  if (!analysis) {
    result.textContent = "—";

    savings.textContent = "—";

    percent.textContent = "—";

    status.textContent = "Noch nicht berechnet";

    return;
  }

  result.textContent = formatBytes(analysis.resultSize);

  savings.textContent = formatBytes(analysis.savingsBytes);

  percent.textContent = `${formatPercent(analysis.savingsPercent)} %`;

  status.textContent = analysis.unchanged
    ? "Mit diesem Preset ist keine weitere Reduktion möglich."
    : `${formatBytes(analysis.originalSize)} → ${formatBytes(analysis.resultSize)}`;
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

  if (input) {
    input.disabled = uploading || processing;
  }

  if (reset) {
    reset.disabled = !upload || uploading || processing;
  }

  if (submit) {
    submit.disabled =
      !upload || !analysis || uploading || analyzing || processing;

    submit.textContent = analysis?.unchanged
      ? "Original herunterladen"
      : "Komprimierte PDF herunterladen";
  }

  buttons?.forEach((button) => {
    button.disabled = uploading || processing;
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

  return `${value.toLocaleString("de-DE", {
    minimumFractionDigits: index === 0 ? 0 : 1,

    maximumFractionDigits: 1,
  })} ${units[index] ?? "B"}`;
}

function formatPercent(value: number): string {
  return value.toLocaleString("de-DE", {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  });
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
