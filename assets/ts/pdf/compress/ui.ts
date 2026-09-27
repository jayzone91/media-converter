import type { PDFUpload } from "../uploads.ts";

import type { PDFCompressionAnalysis, PDFCompressionMode } from "./api.ts";

export interface PDFCompressUIState {
  upload: PDFUpload | null;

  mode: PDFCompressionMode;

  analysis: PDFCompressionAnalysis | null;

  uploading: boolean;

  analyzing: boolean;

  processing: boolean;
}

export function renderCompressUI(
  root: HTMLElement,
  state: PDFCompressUIState,
): void {
  const editor = getElement<HTMLElement>(root, "#pdf-compress-editor");

  const dropZone = getElement<HTMLElement>(root, "#pdf-compress-drop-zone");

  const filename = getElement<HTMLElement>(root, "#pdf-compress-filename");

  const meta = getElement<HTMLElement>(root, "#pdf-compress-meta");

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

  updateCompressModeButtons(root, state.mode);

  updateCompressNotice(root, state.mode);

  renderCompressAnalysis(root, state);

  updateCompressControls(root, state);
}

export function updateCompressModeButtons(
  root: HTMLElement | null,
  mode: PDFCompressionMode,
): void {
  const buttons = root?.querySelectorAll<HTMLButtonElement>(
    "[data-compression-mode]",
  );

  buttons?.forEach((button) => {
    const selected = button.dataset.compressionMode === mode;

    button.classList.toggle("selected", selected);

    button.setAttribute("aria-pressed", selected ? "true" : "false");
  });
}

export function updateCompressNotice(
  root: HTMLElement | null,
  mode: PDFCompressionMode,
): void {
  if (!root) {
    return;
  }

  const notice = getElement<HTMLElement>(root, "#pdf-compress-notice");

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

export function updateCompressControls(
  root: HTMLElement,
  state: PDFCompressUIState,
): void {
  const input = getElement<HTMLInputElement>(root, "#pdf-compress-file");

  const reset = getElement<HTMLButtonElement>(root, "#pdf-compress-reset-file");

  const submit = getElement<HTMLButtonElement>(root, "#pdf-compress-submit");

  const buttons = root.querySelectorAll<HTMLButtonElement>(
    "[data-compression-mode]",
  );

  if (input) {
    input.disabled = state.uploading || state.processing;
  }

  if (reset) {
    reset.disabled = !state.upload || state.uploading || state.processing;
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
      : "Komprimierte PDF herunterladen";
  }

  buttons.forEach((button) => {
    button.disabled = state.uploading || state.processing;
  });
}

export function showCompressError(root: HTMLElement, message: string): void {
  const element = getElement<HTMLElement>(root, "#pdf-compress-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

export function clearCompressError(root: HTMLElement): void {
  const element = getElement<HTMLElement>(root, "#pdf-compress-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}

function renderCompressAnalysis(
  root: HTMLElement,
  state: PDFCompressUIState,
): void {
  const original = getElement<HTMLElement>(root, "#pdf-compress-original-size");

  const result = getElement<HTMLElement>(root, "#pdf-compress-result-size");

  const savings = getElement<HTMLElement>(root, "#pdf-compress-savings-size");

  const percent = getElement<HTMLElement>(
    root,
    "#pdf-compress-savings-percent",
  );

  const status = getElement<HTMLElement>(root, "#pdf-compress-analysis-status");

  if (!original || !result || !savings || !percent || !status) {
    return;
  }

  original.textContent = state.upload ? formatBytes(state.upload.size) : "—";

  if (state.analyzing) {
    result.textContent = "…";

    savings.textContent = "…";

    percent.textContent = "…";

    status.textContent = "Kompression wird berechnet …";

    return;
  }

  if (!state.analysis) {
    result.textContent = "—";

    savings.textContent = "—";

    percent.textContent = "—";

    status.textContent = "Noch nicht berechnet";

    return;
  }

  result.textContent = formatBytes(state.analysis.resultSize);

  savings.textContent = formatBytes(state.analysis.savingsBytes);

  percent.textContent = `${formatPercent(state.analysis.savingsPercent)} %`;

  status.textContent = state.analysis.unchanged
    ? "Mit diesem Preset ist keine weitere Reduktion möglich."
    : `${formatBytes(state.analysis.originalSize)} → ${formatBytes(state.analysis.resultSize)}`;
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

function getElement<T extends Element>(
  root: HTMLElement,
  selector: string,
): T | null {
  return root.querySelector<T>(selector);
}
