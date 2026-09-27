import type { PDFUpload } from "../uploads.ts";

import type { PDFOptimizationAnalysis } from "./api.ts";

export interface PDFOptimizeUIState {
  upload: PDFUpload | null;

  analysis: PDFOptimizationAnalysis | null;

  linearize: boolean;

  uploading: boolean;

  analyzing: boolean;

  processing: boolean;
}

export function renderPDFOptimizeUI(
  root: HTMLElement,
  state: PDFOptimizeUIState,
): void {
  const editor = getElement<HTMLElement>(root, "#pdf-optimize-editor");

  const dropZone = getElement<HTMLElement>(root, "#pdf-optimize-drop-zone");

  const filename = getElement<HTMLElement>(root, "#pdf-optimize-filename");

  const meta = getElement<HTMLElement>(root, "#pdf-optimize-meta");

  const output = getElement<HTMLElement>(root, "#pdf-optimize-output-name");

  if (editor) {
    editor.hidden = state.upload === null;
  }

  if (dropZone) {
    dropZone.hidden = state.upload !== null;
  }

  if (state.upload && filename) {
    filename.textContent = state.upload.filename;
  }

  if (state.upload && meta) {
    meta.textContent = `${state.upload.pageCount} Seiten · ${formatBytes(state.upload.size)}`;
  }

  if (output) {
    output.textContent = state.linearize
      ? "optimiert-web.pdf"
      : "optimiert.pdf";
  }

  renderAnalysis(root, state);

  updateControls(root, state);
}

export function showPDFOptimizeError(root: HTMLElement, message: string): void {
  const element = getElement<HTMLElement>(root, "#pdf-optimize-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

export function clearPDFOptimizeError(root: HTMLElement): void {
  const element = getElement<HTMLElement>(root, "#pdf-optimize-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}

function renderAnalysis(root: HTMLElement, state: PDFOptimizeUIState): void {
  const original = getElement<HTMLElement>(root, "#pdf-optimize-original-size");

  const result = getElement<HTMLElement>(root, "#pdf-optimize-result-size");

  const delta = getElement<HTMLElement>(root, "#pdf-optimize-delta-size");

  const percent = getElement<HTMLElement>(root, "#pdf-optimize-delta-percent");

  const status = getElement<HTMLElement>(root, "#pdf-optimize-analysis-status");

  if (!original || !result || !delta || !percent || !status) {
    return;
  }

  original.textContent = state.upload ? formatBytes(state.upload.size) : "—";

  if (state.analyzing) {
    result.textContent = "…";
    delta.textContent = "…";
    percent.textContent = "…";

    status.textContent = "Optimierung wird berechnet …";

    return;
  }

  if (!state.analysis) {
    result.textContent = "—";
    delta.textContent = "—";
    percent.textContent = "—";

    status.textContent = "Noch nicht berechnet";

    return;
  }

  result.textContent = formatBytes(state.analysis.resultSize);

  delta.textContent = formatSignedBytes(state.analysis.deltaBytes);

  percent.textContent = formatSignedPercent(state.analysis.deltaPercent);

  if (state.analysis.unchanged) {
    status.textContent = "Die PDF ist bereits optimal komprimiert.";
  } else if (state.analysis.linearized) {
    status.textContent =
      "Fast Web View ist aktiviert. Eine leichte Größenzunahme ist dabei möglich.";
  } else {
    status.textContent = "PDF-Struktur wurde verlustfrei optimiert.";
  }
}

function updateControls(root: HTMLElement, state: PDFOptimizeUIState): void {
  const input = getElement<HTMLInputElement>(root, "#pdf-optimize-file");

  const reset = getElement<HTMLButtonElement>(root, "#pdf-optimize-reset-file");

  const linearize = getElement<HTMLInputElement>(
    root,
    "#pdf-optimize-linearize",
  );

  const submit = getElement<HTMLButtonElement>(root, "#pdf-optimize-submit");

  const busy = state.uploading || state.processing;

  if (input) {
    input.disabled = busy;
  }

  if (reset) {
    reset.disabled = !state.upload || busy;
  }

  if (linearize) {
    linearize.disabled = !state.upload || busy;
  }

  if (submit) {
    submit.disabled =
      !state.upload ||
      !state.analysis ||
      state.uploading ||
      state.analyzing ||
      state.processing;

    submit.textContent = state.analysis?.unchanged
      ? "Original herunterladen"
      : "Optimierte PDF herunterladen";
  }
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

function formatSignedBytes(bytes: number): string {
  if (bytes === 0) {
    return "0 B";
  }

  const prefix = bytes > 0 ? "+" : "−";

  return `${prefix}${formatBytes(Math.abs(bytes))}`;
}

function formatSignedPercent(value: number): string {
  if (value === 0) {
    return "0,0 %";
  }

  const prefix = value > 0 ? "+" : "−";

  return `${prefix}${Math.abs(value).toLocaleString("de-DE", {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })} %`;
}

function getElement<T extends Element>(
  root: HTMLElement,
  selector: string,
): T | null {
  return root.querySelector<T>(selector);
}
