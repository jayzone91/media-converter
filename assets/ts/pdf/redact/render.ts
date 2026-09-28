import {
  findRedaction,
  getPageRedactions,
  hasRedactions,
  pageHasRedactions,
  redactState,
  type PDFRedaction,
} from "./state.ts";

export function renderRedactions(): void {
  const overlay = redactOverlay();

  if (!overlay) {
    return;
  }

  overlay.replaceChildren();

  for (const redaction of getPageRedactions(redactState.activePage)) {
    overlay.append(createRedactionElement(redaction));
  }
}

export function updateControls(): void {
  const root = redactState.root;

  if (!root) {
    return;
  }

  const properties = root.querySelector<HTMLElement>("#pdf-redact-properties");

  const deleteButton =
    root.querySelector<HTMLButtonElement>("#pdf-redact-delete");

  const clearButton = root.querySelector<HTMLButtonElement>(
    "#pdf-redact-clear-page",
  );

  const selectionInfo = root.querySelector<HTMLElement>(
    "#pdf-redact-selection-info",
  );

  const applyButton =
    root.querySelector<HTMLButtonElement>("#pdf-redact-apply");

  const selected = redactState.selectedID
    ? findRedaction(redactState.activePage, redactState.selectedID)
    : null;

  if (properties) {
    properties.hidden = !selected;
  }

  if (deleteButton) {
    deleteButton.disabled = !selected;
  }

  if (clearButton) {
    clearButton.disabled =
      getPageRedactions(redactState.activePage).length === 0;
  }

  if (selectionInfo && selected) {
    selectionInfo.textContent = `${Math.round(
      selected.width * 100,
    )} × ${Math.round(selected.height * 100)} %`;
  }

  if (applyButton) {
    applyButton.disabled = true;

    applyButton.dataset.ready = hasRedactions() ? "true" : "false";
  }
}

export function updatePageIndicators(): void {
  const root = redactState.root;

  if (!root) {
    return;
  }

  root
    .querySelectorAll<HTMLButtonElement>(".pdf-edit-page")
    .forEach((button) => {
      const page = Number(button.dataset.page);

      const indicator = button.querySelector<HTMLElement>(
        ".pdf-edit-page-change-indicator",
      );

      if (!indicator) {
        return;
      }

      indicator.hidden = !pageHasRedactions(page);
    });
}

export function deleteSelectedRedaction(): void {
  const id = redactState.selectedID;

  if (!id) {
    return;
  }

  removeRedaction(id);
}

export function removeRedaction(id: string): void {
  const redactions = getPageRedactions(redactState.activePage).filter(
    (redaction) => redaction.id !== id,
  );

  if (redactions.length === 0) {
    redactState.pageRedactions.delete(redactState.activePage);
  } else {
    redactState.pageRedactions.set(redactState.activePage, redactions);
  }

  if (redactState.selectedID === id) {
    redactState.selectedID = null;
  }

  renderRedactions();
  updateControls();
  updatePageIndicators();
}

export function clearCurrentPage(): void {
  redactState.pageRedactions.delete(redactState.activePage);

  redactState.selectedID = null;

  renderRedactions();
  updateControls();
  updatePageIndicators();
}

function createRedactionElement(redaction: PDFRedaction): HTMLButtonElement {
  const element = document.createElement("button");

  element.type = "button";

  element.className = "pdf-redact-area";

  element.dataset.redactionId = redaction.id;

  element.setAttribute("aria-label", "Schwärzungsbereich auswählen");

  element.style.left = `${redaction.x * 100}%`;

  element.style.top = `${redaction.y * 100}%`;

  element.style.width = `${redaction.width * 100}%`;

  element.style.height = `${redaction.height * 100}%`;

  element.classList.toggle("selected", redactState.selectedID === redaction.id);

  if (redactState.selectedID === redaction.id) {
    const handle = document.createElement("span");

    handle.className = "pdf-redact-resize-handle";

    handle.setAttribute("aria-hidden", "true");

    element.append(handle);
  }

  return element;
}

function redactOverlay(): HTMLElement | null {
  return (
    redactState.root?.querySelector<HTMLElement>("#pdf-redact-overlay") ?? null
  );
}
