import { downloadBlob } from "../../shared/download.ts";

import { createPDFDocument, type PDFCreatePaperSize } from "./api.ts";

let root: HTMLElement | null = null;

let processing = false;

export function setupPDFCreate(workspace: HTMLElement): void {
  root = workspace;
  processing = false;

  const title = getElement<HTMLInputElement>("#pdf-create-title");

  const body = getElement<HTMLTextAreaElement>("#pdf-create-body");

  const markdown = getElement<HTMLInputElement>("#pdf-create-markdown");

  const submit = getElement<HTMLButtonElement>("#pdf-create-submit");

  if (!title || !body || !markdown || !submit) {
    return;
  }

  title.addEventListener("input", updateControls);

  body.addEventListener("input", updateControls);

  markdown.addEventListener("change", () => {
    updateEditorMode();
    updateControls();
  });

  submit.addEventListener("click", () => {
    void createDocument();
  });

  updateEditorMode();
  updateControls();
}

export function destroyPDFCreate(): void {
  root = null;
  processing = false;
}

async function createDocument(): Promise<void> {
  if (!root || processing) {
    return;
  }

  const title = getElement<HTMLInputElement>("#pdf-create-title");

  const body = getElement<HTMLTextAreaElement>("#pdf-create-body");

  const paper = getElement<HTMLSelectElement>("#pdf-create-paper-size");

  const fontSize = getElement<HTMLSelectElement>("#pdf-create-font-size");

  const landscape = getElement<HTMLInputElement>("#pdf-create-landscape");

  const includeDate = getElement<HTMLInputElement>("#pdf-create-date");

  const markdown = getElement<HTMLInputElement>("#pdf-create-markdown");

  if (
    !title ||
    !body ||
    !paper ||
    !fontSize ||
    !landscape ||
    !includeDate ||
    !markdown
  ) {
    return;
  }

  if (title.value.trim() === "" && body.value.trim() === "") {
    showError("Bitte einen Titel oder Dokumentinhalt eingeben.");

    return;
  }

  const paperSize = readPaperSize(paper.value);

  if (!paperSize) {
    showError("Ungültiges Papierformat.");

    return;
  }

  const parsedFontSize = Number.parseInt(fontSize.value, 10);

  if (
    !Number.isInteger(parsedFontSize) ||
    parsedFontSize < 10 ||
    parsedFontSize > 24
  ) {
    showError("Ungültige Schriftgröße.");

    return;
  }

  clearError();

  processing = true;
  updateControls();

  const progress = getElement<HTMLElement>("#pdf-create-progress");

  if (progress) {
    progress.hidden = false;
  }

  try {
    const result = await createPDFDocument({
      title: title.value.trim(),

      body: body.value.trim(),

      paperSize,

      landscape: landscape.checked,

      fontSize: parsedFontSize,

      includeDate: includeDate.checked,

      markdown: markdown.checked,
    });

    downloadBlob(result.blob, result.filename);
  } catch (error: unknown) {
    showError(
      error instanceof Error
        ? error.message
        : "Das PDF-Dokument konnte nicht erstellt werden.",
    );
  } finally {
    processing = false;

    if (progress) {
      progress.hidden = true;
    }

    updateControls();
  }
}

function updateEditorMode(): void {
  const body = getElement<HTMLTextAreaElement>("#pdf-create-body");

  const markdown = getElement<HTMLInputElement>("#pdf-create-markdown");

  if (!body || !markdown) {
    return;
  }

  body.placeholder = markdown.checked
    ? "# Überschrift\n\nText mit **fett**, *kursiv*, Listen, Tabellen oder Code …"
    : "Text des Dokuments …";

  body.classList.toggle("pdf-create-markdown-input", markdown.checked);
}

function updateControls(): void {
  const title = getElement<HTMLInputElement>("#pdf-create-title");

  const body = getElement<HTMLTextAreaElement>("#pdf-create-body");

  const submit = getElement<HTMLButtonElement>("#pdf-create-submit");

  const controls = root?.querySelectorAll<
    HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement
  >("input, textarea, select");

  controls?.forEach((control) => {
    control.disabled = processing;
  });

  if (submit) {
    submit.disabled =
      processing || (title?.value.trim() === "" && body?.value.trim() === "");
  }
}

function readPaperSize(value: string): PDFCreatePaperSize | null {
  switch (value) {
    case "a4":
      return "a4";

    case "letter":
      return "letter";

    default:
      return null;
  }
}

function showError(message: string): void {
  const element = getElement<HTMLElement>("#pdf-create-error");

  if (!element) {
    return;
  }

  element.textContent = message;
  element.hidden = false;
}

function clearError(): void {
  const element = getElement<HTMLElement>("#pdf-create-error");

  if (!element) {
    return;
  }

  element.textContent = "";
  element.hidden = true;
}

function getElement<T extends Element>(selector: string): T | null {
  return root?.querySelector<T>(selector) ?? null;
}
