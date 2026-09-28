import {
  clamp,
  createTextObject,
  editorState,
  findTextObject,
  getPageTextObjects,
  pageHasEdits,
  type PDFTextObject,
} from "./state.ts";

import { updatePDFEditSubmitState } from "./submit.ts";

const pdfPreviewDPI = 90;
const pdfPointsPerInch = 72;

export function setupPDFEditText(): boolean {
  const root = editorState.root;

  if (!root) {
    return false;
  }

  const addTextButton =
    root.querySelector<HTMLButtonElement>("#pdf-edit-add-text");

  const textContent = root.querySelector<HTMLInputElement>(
    "#pdf-edit-text-content",
  );

  const textSize = root.querySelector<HTMLInputElement>("#pdf-edit-text-size");

  const textColor = root.querySelector<HTMLInputElement>(
    "#pdf-edit-text-color",
  );

  const deleteText = root.querySelector<HTMLButtonElement>(
    "#pdf-edit-delete-text",
  );

  if (
    !addTextButton ||
    !textContent ||
    !textSize ||
    !textColor ||
    !deleteText
  ) {
    return false;
  }

  addTextButton.addEventListener("click", addTextObject);

  textContent.addEventListener("input", () => {
    updateSelectedText({
      text: textContent.value,
    });
  });

  textSize.addEventListener("input", () => {
    const value = Number(textSize.value);

    if (!Number.isFinite(value) || value < 6 || value > 144) {
      return;
    }

    updateSelectedText({
      size: value,
    });
  });

  textColor.addEventListener("input", () => {
    updateSelectedText({
      color: textColor.value,
    });
  });

  deleteText.addEventListener("click", deleteSelectedText);

  return true;
}

export function renderTextObjects(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const overlay = root.querySelector<HTMLElement>("#pdf-edit-overlay");

  if (!overlay) {
    return;
  }

  overlay.replaceChildren();

  const objects = getPageTextObjects(editorState.activePage);

  for (const object of objects) {
    overlay.append(createTextElement(object));
  }
}

export function updateTextProperties(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const properties = root.querySelector<HTMLElement>(
    "#pdf-edit-text-properties",
  );

  const content = root.querySelector<HTMLInputElement>(
    "#pdf-edit-text-content",
  );

  const size = root.querySelector<HTMLInputElement>("#pdf-edit-text-size");

  const color = root.querySelector<HTMLInputElement>("#pdf-edit-text-color");

  if (!properties || !content || !size || !color) {
    return;
  }

  const object = editorState.selectedTextID
    ? findTextObject(editorState.activePage, editorState.selectedTextID)
    : null;

  if (!object) {
    properties.hidden = true;

    return;
  }

  properties.hidden = false;

  if (content.value !== object.text) {
    content.value = object.text;
  }

  const sizeValue = String(object.size);

  if (size.value !== sizeValue) {
    size.value = sizeValue;
  }

  if (color.value !== object.color) {
    color.value = object.color;
  }
}

export function updatePageChangeIndicators(): void {
  const root = editorState.root;

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

      indicator.hidden = !pageHasEdits(page);
    });
}

function addTextObject(): void {
  if (!editorState.root || !editorState.activeUpload) {
    return;
  }

  const object = createTextObject();

  const objects = getPageTextObjects(editorState.activePage);

  objects.push(object);

  editorState.pageTextObjects.set(editorState.activePage, objects);

  editorState.selectedTextID = object.id;

  editorState.selectedImageID = null;

  hideImageProperties();

  renderTextObjects();
  updateTextProperties();
  updatePageChangeIndicators();
  updatePDFEditSubmitState();
}

function createTextElement(object: PDFTextObject): HTMLButtonElement {
  const element = document.createElement("button");

  element.type = "button";

  element.className = "pdf-edit-text-object";

  element.dataset.textId = object.id;

  element.textContent = object.text || " ";

  element.style.left = `${object.x * 100}%`;

  element.style.top = `${object.y * 100}%`;

  element.style.fontSize = `${previewFontSize(object.size)}px`;

  element.style.color = object.color;

  element.classList.toggle(
    "selected",
    object.id === editorState.selectedTextID,
  );

  element.setAttribute("aria-label", `Textobjekt ${object.text || "leer"}`);

  element.addEventListener("click", (event) => {
    event.stopPropagation();

    selectTextObject(object.id);
  });

  element.addEventListener("pointerdown", (event) => {
    beginTextDrag(event, object.id, element);
  });

  element.addEventListener("pointermove", moveTextDrag);

  element.addEventListener("pointerup", endTextDrag);

  element.addEventListener("pointercancel", endTextDrag);

  return element;
}

function previewFontSize(points: number): number {
  const root = editorState.root;

  if (!root) {
    return points;
  }

  const preview = root.querySelector<HTMLImageElement>(
    "#pdf-edit-page-preview",
  );

  if (!preview) {
    return points;
  }

  const naturalWidth = preview.naturalWidth;

  const displayedWidth = preview.getBoundingClientRect().width;

  const previewPixelsPerPoint = pdfPreviewDPI / pdfPointsPerInch;

  if (naturalWidth <= 0 || displayedWidth <= 0) {
    return points * previewPixelsPerPoint;
  }

  const displayScale = displayedWidth / naturalWidth;

  return points * previewPixelsPerPoint * displayScale;
}

function selectTextObject(id: string): void {
  editorState.selectedTextID = id;

  editorState.selectedImageID = null;

  hideImageProperties();

  renderTextObjects();
  updateTextProperties();
}

function beginTextDrag(
  event: PointerEvent,
  id: string,
  element: HTMLElement,
): void {
  event.preventDefault();

  const rectangle = element.getBoundingClientRect();

  editorState.selectedTextID = id;

  editorState.selectedImageID = null;

  hideImageProperties();

  editorState.dragState = {
    id,

    pointerId: event.pointerId,

    offsetX: event.clientX - rectangle.left,

    offsetY: event.clientY - rectangle.top,
  };

  element.setPointerCapture(event.pointerId);

  updateTextProperties();
}

function moveTextDrag(event: PointerEvent): void {
  const root = editorState.root;

  const drag = editorState.dragState;

  if (!root || !drag || drag.pointerId !== event.pointerId) {
    return;
  }

  const element = event.currentTarget;

  if (!(element instanceof HTMLElement)) {
    return;
  }

  const overlay = root.querySelector<HTMLElement>("#pdf-edit-overlay");

  if (!overlay) {
    return;
  }

  const rectangle = overlay.getBoundingClientRect();

  if (rectangle.width <= 0 || rectangle.height <= 0) {
    return;
  }

  const x = (event.clientX - rectangle.left - drag.offsetX) / rectangle.width;

  const y = (event.clientY - rectangle.top - drag.offsetY) / rectangle.height;

  setTextPosition(drag.id, x, y, element);
}

function endTextDrag(event: PointerEvent): void {
  const drag = editorState.dragState;

  if (!drag || drag.pointerId !== event.pointerId) {
    return;
  }

  const element = event.currentTarget;

  if (
    element instanceof HTMLElement &&
    element.hasPointerCapture(event.pointerId)
  ) {
    element.releasePointerCapture(event.pointerId);
  }

  editorState.dragState = null;
}

function setTextPosition(
  id: string,
  x: number,
  y: number,
  element: HTMLElement,
): void {
  const object = findTextObject(editorState.activePage, id);

  if (!object) {
    return;
  }

  const parentWidth = Math.max(1, element.parentElement?.clientWidth ?? 1);

  const parentHeight = Math.max(1, element.parentElement?.clientHeight ?? 1);

  const maxX = Math.max(0, 1 - element.offsetWidth / parentWidth);

  const maxY = Math.max(0, 1 - element.offsetHeight / parentHeight);

  object.x = clamp(x, 0, maxX);

  object.y = clamp(y, 0, maxY);

  element.style.left = `${object.x * 100}%`;

  element.style.top = `${object.y * 100}%`;
}

function updateSelectedText(
  changes: Partial<Pick<PDFTextObject, "text" | "size" | "color">>,
): void {
  const id = editorState.selectedTextID;

  if (!id) {
    return;
  }

  const object = findTextObject(editorState.activePage, id);

  if (!object) {
    return;
  }

  if (changes.text !== undefined) {
    object.text = changes.text;
  }

  if (changes.size !== undefined) {
    object.size = changes.size;
  }

  if (changes.color !== undefined) {
    object.color = changes.color;
  }

  renderTextObjects();
  updateTextProperties();
  updatePageChangeIndicators();
  updatePDFEditSubmitState();
}

function deleteSelectedText(): void {
  const id = editorState.selectedTextID;

  if (!id) {
    return;
  }

  const objects = getPageTextObjects(editorState.activePage);

  const filtered = objects.filter((object) => object.id !== id);

  editorState.pageTextObjects.set(editorState.activePage, filtered);

  editorState.selectedTextID = null;

  renderTextObjects();
  updateTextProperties();
  updatePageChangeIndicators();
  updatePDFEditSubmitState();
}

function hideImageProperties(): void {
  const properties = editorState.root?.querySelector<HTMLElement>(
    "#pdf-edit-image-properties",
  );

  if (properties) {
    properties.hidden = true;
  }
}
