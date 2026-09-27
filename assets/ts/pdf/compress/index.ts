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

import { setupPDFCompressDropZone } from "./drop-zone.ts";

import {
  clearCompressError,
  renderCompressUI,
  showCompressError,
  updateCompressModeButtons,
  updateCompressNotice,
  type PDFCompressUIState,
} from "./ui.ts";

import { downloadURL } from "../../shared/download.ts";

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

  resetState();

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

  setupPDFCompressDropZone(
    dropZone,
    (file) => {
      void selectFile(file);
    },
    (message) => {
      showError(message);
    },
  );

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

  resetState();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

function resetState(): void {
  upload = null;

  mode = "balanced";

  analysis = null;

  analysisCache.clear();

  analysisController = null;

  analysisRequestID = 0;

  uploading = false;

  analyzing = false;

  processing = false;
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

      updateCompressModeButtons(root, mode);

      updateCompressNotice(root, mode);

      if (!upload) {
        return;
      }

      const cached = analysisCache.get(mode);

      if (cached) {
        analysisController?.abort();

        analysis = cached;

        analyzing = false;

        render();

        return;
      }

      void analyzeCurrentMode();
    });
  }
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

  render();

  const progress = getElement<HTMLElement>("#pdf-compress-upload-progress");

  if (progress) {
    progress.hidden = false;
  }

  let uploaded = false;

  try {
    upload = await uploadPDF(file);

    uploaded = true;
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

    render();
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

    render();

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

  render();

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

      render();
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

  render();

  const progress = getElement<HTMLElement>("#pdf-compress-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await compressPDF(upload.id, mode);

    downloadURL(result.downloadURL);

    upload = null;

    analysis = null;

    analysisCache.clear();
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

    render();
  }
}

function render(): void {
  if (!root) {
    return;
  }

  renderCompressUI(root, currentUIState());
}

function currentUIState(): PDFCompressUIState {
  return {
    upload,
    mode,
    analysis,
    uploading,
    analyzing,
    processing,
  };
}

function showError(message: string): void {
  if (!root) {
    return;
  }

  showCompressError(root, message);
}

function clearError(): void {
  if (!root) {
    return;
  }

  clearCompressError(root);
}

function isCompressionMode(
  value: string | undefined,
): value is PDFCompressionMode {
  return value === "lossless" || value === "balanced" || value === "strong";
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
