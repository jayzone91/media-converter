import { downloadBlob } from "../../shared/download.ts";

import {
  deletePDFUpload,
  isPDFFile,
  type PDFUpload,
  uploadPDF,
} from "../uploads.ts";

import {
  analyzePDFOptimization,
  optimizePDF,
  type PDFOptimizationAnalysis,
} from "./api.ts";

import {
  clearPDFOptimizeError,
  renderPDFOptimizeUI,
  showPDFOptimizeError,
  type PDFOptimizeUIState,
} from "./ui.ts";

const MAX_FILE_SIZE = 512 * 1024 * 1024;

let root: HTMLElement | null = null;

let upload: PDFUpload | null = null;

let analysis: PDFOptimizationAnalysis | null = null;

let linearize = false;

const analysisCache = new Map<boolean, PDFOptimizationAnalysis>();

let analysisController: AbortController | null = null;

let analysisRequestID = 0;

let uploading = false;

let analyzing = false;

let processing = false;

export function setupPDFOptimize(workspace: HTMLElement): void {
  root = workspace;

  resetState();

  const input = getElement<HTMLInputElement>("#pdf-optimize-file");

  const dropZone = getElement<HTMLElement>("#pdf-optimize-drop-zone");

  const reset = getElement<HTMLButtonElement>("#pdf-optimize-reset-file");

  const linearizeInput = getElement<HTMLInputElement>(
    "#pdf-optimize-linearize",
  );

  const submit = getElement<HTMLButtonElement>("#pdf-optimize-submit");

  if (!input || !dropZone || !reset || !linearizeInput || !submit) {
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

  reset.addEventListener("click", () => {
    void resetUpload();
  });

  linearizeInput.addEventListener("change", () => {
    linearize = linearizeInput.checked;

    const cached = analysisCache.get(linearize);

    if (cached) {
      analysisController?.abort();

      analysis = cached;

      analyzing = false;

      render();

      return;
    }

    void analyzeCurrentMode();
  });

  submit.addEventListener("click", () => {
    void downloadOptimizedPDF();
  });

  render();
}

export async function destroyPDFOptimize(): Promise<void> {
  analysisController?.abort();

  const uploadID = upload?.id;

  root = null;

  resetState();

  if (uploadID) {
    await deletePDFUpload(uploadID);
  }
}

function resetState(): void {
  upload = null;

  analysis = null;

  linearize = false;

  analysisCache.clear();

  analysisController = null;

  analysisRequestID = 0;

  uploading = false;

  analyzing = false;

  processing = false;
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
      showError("Es kann nur eine PDF gleichzeitig optimiert werden.");

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

  const validation = validateFile(file);

  if (validation) {
    showError(validation);

    return;
  }

  analysisController?.abort();

  analysis = null;

  analysisCache.clear();

  if (upload) {
    await deletePDFUpload(upload.id);

    upload = null;
  }

  uploading = true;

  render();

  const progress = getElement<HTMLElement>("#pdf-optimize-upload-progress");

  if (progress) {
    progress.hidden = false;
  }

  let uploaded = false;

  try {
    upload = await uploadPDF(file);

    uploaded = true;
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "PDF konnte nicht hochgeladen werden.",
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

  const cached = analysisCache.get(linearize);

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

  const requestedLinearize = linearize;

  analysis = null;

  analyzing = true;

  clearError();

  render();

  try {
    const result = await analyzePDFOptimization(
      uploadID,
      requestedLinearize,
      controller.signal,
    );

    if (
      requestID !== analysisRequestID ||
      !upload ||
      upload.id !== uploadID ||
      linearize !== requestedLinearize
    ) {
      return;
    }

    analysisCache.set(requestedLinearize, result);

    analysis = result;
  } catch (error: unknown) {
    if (error instanceof DOMException && error.name === "AbortError") {
      return;
    }

    if (requestID === analysisRequestID) {
      showError(
        error instanceof Error
          ? error.message
          : "Die Optimierung konnte nicht berechnet werden.",
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

async function downloadOptimizedPDF(): Promise<void> {
  if (!upload || !analysis || uploading || analyzing || processing) {
    return;
  }

  clearError();

  processing = true;

  render();

  const progress = getElement<HTMLElement>("#pdf-optimize-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await optimizePDF(upload.id, linearize);

    downloadBlob(result.blob, result.filename);

    upload = null;

    analysis = null;

    analysisCache.clear();
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Die optimierte PDF konnte nicht heruntergeladen werden.",
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

  renderPDFOptimizeUI(root, currentUIState());
}

function currentUIState(): PDFOptimizeUIState {
  return {
    upload,
    analysis,
    linearize,
    uploading,
    analyzing,
    processing,
  };
}

function validateFile(file: File): string | null {
  if (!isPDFFile(file)) {
    return "Es können nur PDF-Dateien hochgeladen werden.";
  }

  if (file.size <= 0) {
    return "Die PDF-Datei ist leer.";
  }

  if (file.size > MAX_FILE_SIZE) {
    return "Die PDF-Datei ist größer als 512 MiB.";
  }

  return null;
}

function showError(message: string): void {
  if (root) {
    showPDFOptimizeError(root, message);
  }
}

function clearError(): void {
  if (root) {
    clearPDFOptimizeError(root);
  }
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
