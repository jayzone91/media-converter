import { downloadURL } from "../../shared/download.ts";

import {
  createWebPDF,
  type PDFWebPaperSize,
  type PDFWebRenderMode,
} from "./api.ts";

let root: HTMLElement | null = null;

let processing = false;

export function setupPDFWeb(workspace: HTMLElement): void {
  root = workspace;

  processing = false;

  const url = getElement<HTMLInputElement>("#pdf-web-url");

  const submit = getElement<HTMLButtonElement>("#pdf-web-submit");

  if (!url || !submit) {
    return;
  }

  url.addEventListener("input", updateControls);

  url.addEventListener("keydown", (event) => {
    if (event.key === "Enter" && !submit.disabled) {
      event.preventDefault();

      void createPDF();
    }
  });

  submit.addEventListener("click", () => {
    void createPDF();
  });

  updateControls();
}

export function destroyPDFWeb(): void {
  root = null;

  processing = false;
}

async function createPDF(): Promise<void> {
  if (!root || processing) {
    return;
  }

  const url = getElement<HTMLInputElement>("#pdf-web-url");

  const paper = getElement<HTMLSelectElement>("#pdf-web-paper-size");

  const wait = getElement<HTMLSelectElement>("#pdf-web-wait");

  const landscape = getElement<HTMLInputElement>("#pdf-web-landscape");

  const background = getElement<HTMLInputElement>("#pdf-web-background");

  const renderMode = getSelectedRenderMode();

  if (!url || !paper || !wait || !landscape || !background || !renderMode) {
    return;
  }

  const normalizedURL = url.value.trim();

  if (!isValidURL(normalizedURL)) {
    showError("Bitte eine vollständige HTTP- oder HTTPS-Adresse eingeben.");

    return;
  }

  const paperSize = readPaperSize(paper.value);

  if (!paperSize) {
    showError("Ungültiges Papierformat.");

    return;
  }

  const waitMilliseconds = Number.parseInt(wait.value, 10);

  if (
    !Number.isInteger(waitMilliseconds) ||
    waitMilliseconds < 0 ||
    waitMilliseconds > 10000
  ) {
    showError("Ungültige Wartezeit.");

    return;
  }

  clearError();

  processing = true;

  updateControls();

  const progress = getElement<HTMLElement>("#pdf-web-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await createWebPDF({
      url: normalizedURL,

      paperSize,

      renderMode,

      landscape: landscape.checked,

      printBackground: background.checked,

      waitMilliseconds,
    });

    downloadURL(result.downloadURL);
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Die Webseite konnte nicht als PDF erstellt werden.",
    );
  } finally {
    processing = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

function updateControls(): void {
  const url = getElement<HTMLInputElement>("#pdf-web-url");

  const submit = getElement<HTMLButtonElement>("#pdf-web-submit");

  const controls = root?.querySelectorAll<HTMLInputElement | HTMLSelectElement>(
    "input, select",
  );

  controls?.forEach((control) => {
    control.disabled = processing;
  });

  if (submit) {
    submit.disabled = processing || !url || url.value.trim() === "";
  }
}

function getSelectedRenderMode(): PDFWebRenderMode | null {
  const selected = root?.querySelector<HTMLInputElement>(
    'input[name="pdf-web-render-mode"]:checked',
  );

  if (!selected) {
    return null;
  }

  return readRenderMode(selected.value);
}

function isValidURL(value: string): boolean {
  try {
    const parsed = new URL(value);

    return (
      (parsed.protocol === "http:" || parsed.protocol === "https:") &&
      parsed.hostname !== "" &&
      parsed.username === "" &&
      parsed.password === ""
    );
  } catch {
    return false;
  }
}

function readPaperSize(value: string): PDFWebPaperSize | null {
  switch (value) {
    case "a4":
      return "a4";

    case "letter":
      return "letter";

    default:
      return null;
  }
}

function readRenderMode(value: string): PDFWebRenderMode | null {
  switch (value) {
    case "desktop":
      return "desktop";

    case "tablet":
      return "tablet";

    case "mobile":
      return "mobile";

    case "print":
      return "print";

    default:
      return null;
  }
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-web-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-web-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
