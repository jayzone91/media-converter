import { buildQRRequest, hasQRPayload } from "./request.ts";

import type { QRResponse } from "./types.ts";

let previewTimeout: number | null = null;

let previewController: AbortController | null = null;

let lastGeneratedSVG = "";

export function scheduleQRPreview(): void {
  if (previewTimeout !== null) {
    window.clearTimeout(previewTimeout);
  }

  previewTimeout = window.setTimeout(() => {
    void generateQRPreview();
  }, 250);
}

export function getLastGeneratedQRSVG(): string {
  return lastGeneratedSVG;
}

export function showQRStatus(text: string): void {
  const status = document.querySelector<HTMLElement>(".qr-status");

  if (status) {
    status.textContent = text;
  }
}

export async function generateQRPreview(): Promise<void> {
  const preview = document.querySelector<HTMLElement>("#qr-preview");

  if (!preview) {
    return;
  }

  const errorLevel = document.querySelector<HTMLElement>("#qr-error-level");

  const version = document.querySelector<HTMLElement>("#qr-version");

  const request = buildQRRequest();

  if (!hasQRPayload(request)) {
    lastGeneratedSVG = "";

    showEmptyQRPreview(preview);

    setDownloadVisibility(false);

    if (errorLevel) {
      errorLevel.textContent = "L";
    }

    if (version) {
      version.textContent = "Auto";
    }

    showQRStatus("Bereit");

    return;
  }

  previewController?.abort();

  const controller = new AbortController();

  previewController = controller;

  showQRStatus("Erzeuge …");

  try {
    const response = await fetch("/qr/generate", {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify(request),

      signal: controller.signal,
    });

    if (!response.ok) {
      const message = await response.text();

      throw new Error(message.trim() || "QR Code konnte nicht erzeugt werden.");
    }

    const result = (await response.json()) as QRResponse;

    lastGeneratedSVG = result.svg;

    preview.innerHTML = result.svg;

    const svg = preview.querySelector<SVGElement>("svg");

    if (svg) {
      svg.style.width = "100%";

      svg.style.height = "100%";

      svg.style.display = "block";
    }

    if (errorLevel) {
      errorLevel.textContent = result.error_correction;
    }

    if (version) {
      version.textContent = String(result.version);
    }

    setDownloadVisibility(true);

    showQRStatus("Aktuell");
  } catch (error: unknown) {
    if (error instanceof DOMException && error.name === "AbortError") {
      return;
    }

    console.error("QR preview failed:", error);

    lastGeneratedSVG = "";

    setDownloadVisibility(false);

    showQRStatus("Fehler");

    showQRPreviewError(
      preview,
      error instanceof Error
        ? error.message
        : "QR Code konnte nicht erzeugt werden.",
    );
  } finally {
    if (previewController === controller) {
      previewController = null;
    }
  }
}

function setDownloadVisibility(visible: boolean): void {
  const container = document.querySelector<HTMLElement>(".qr-download-actions");

  if (container) {
    container.hidden = !visible;
  }
}

function showEmptyQRPreview(preview: HTMLElement): void {
  preview.innerHTML = `
    <div class="qr-preview-placeholder">
      <div class="preview-icon">
        ▦
      </div>

      <strong>
        QR Code Vorschau
      </strong>

      <span>
        Gib links einen Inhalt ein.
      </span>
    </div>
  `;
}

function showQRPreviewError(preview: HTMLElement, message: string): void {
  preview.innerHTML = `
    <div class="qr-preview-placeholder">
      <strong>
        Keine Vorschau
      </strong>

      <span>
        ${escapeHTML(message)}
      </span>
    </div>
  `;
}

function escapeHTML(value: string): string {
  const element = document.createElement("div");

  element.textContent = value;

  return element.innerHTML;
}
