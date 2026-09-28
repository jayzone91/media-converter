import { hasRedactions, redactState } from "./state.ts";

import { showRedactResult, type PDFRedactResult } from "./result.ts";

interface RedactionRequest {
  page: number;

  x: number;
  y: number;

  width: number;
  height: number;
}

interface PDFRedactRequest {
  upload_id: string;

  redactions: RedactionRequest[];
}

export function setupRedactSubmit(): boolean {
  const root = redactState.root;

  if (!root) {
    return false;
  }

  const button = root.querySelector<HTMLButtonElement>("#pdf-redact-apply");

  if (!button) {
    return false;
  }

  button.addEventListener("click", () => {
    void submitRedactions();
  });

  updateRedactSubmitState();

  return true;
}

export function updateRedactSubmitState(): void {
  const button =
    redactState.root?.querySelector<HTMLButtonElement>("#pdf-redact-apply");

  if (!button) {
    return;
  }

  button.disabled = !redactState.activeUpload || !hasRedactions();
}

async function submitRedactions(): Promise<void> {
  const root = redactState.root;

  const upload = redactState.activeUpload;

  if (!root || !upload) {
    return;
  }

  const button = root.querySelector<HTMLButtonElement>("#pdf-redact-apply");

  const error = root.querySelector<HTMLElement>("#pdf-redact-error");

  if (!button) {
    return;
  }

  const redactions = buildRedactionRequest();

  if (redactions.length === 0) {
    return;
  }

  if (error) {
    error.hidden = true;
    error.textContent = "";
  }

  const request: PDFRedactRequest = {
    upload_id: upload.id,

    redactions,
  };

  const originalText = button.textContent;

  button.disabled = true;

  button.textContent = "Schwärzung wird angewendet …";

  try {
    const response = await fetch("/pdf/redact", {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify(request),
    });

    if (!response.ok) {
      const message = await response.text();

      throw new Error(message.trim() || "PDF konnte nicht geschwärzt werden.");
    }

    const data = (await response.json()) as unknown;

    if (!isPDFRedactResult(data)) {
      throw new Error("Der Server hat eine ungültige Antwort geliefert.");
    }

    redactState.activeUpload = null;

    showRedactResult(data);
  } catch (caught: unknown) {
    if (error) {
      error.textContent = errorMessage(caught);

      error.hidden = false;
    }
  } finally {
    button.textContent = originalText;

    updateRedactSubmitState();
  }
}

function buildRedactionRequest(): RedactionRequest[] {
  const result: RedactionRequest[] = [];

  for (const [page, redactions] of redactState.pageRedactions) {
    for (const redaction of redactions) {
      result.push({
        page: page + 1,

        x: redaction.x,

        y: redaction.y,

        width: redaction.width,

        height: redaction.height,
      });
    }
  }

  return result;
}

function isPDFRedactResult(value: unknown): value is PDFRedactResult {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const candidate = value as Record<string, unknown>;

  return (
    typeof candidate.download_url === "string" &&
    typeof candidate.filename === "string" &&
    typeof candidate.preview_upload_id === "string" &&
    Array.isArray(candidate.previews) &&
    candidate.previews.every((preview) => typeof preview === "string") &&
    typeof candidate.page_count === "number"
  );
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return "PDF konnte nicht geschwärzt werden.";
}
