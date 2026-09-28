import {
  clamp,
  createImageObject,
  editorState,
  findImageObject,
  getPageImageObjects,
} from "./state.ts";

import { updatePDFEditSubmitState } from "./submit.ts";

import { updatePageChangeIndicators } from "./text.ts";

const allowedImageTypes = new Set(["image/jpeg", "image/png"]);

const maxImageSize = 32 * 1024 * 1024;

export function setupPDFEditImage(): boolean {
  const root = editorState.root;

  if (!root) {
    return false;
  }

  const button = root.querySelector<HTMLButtonElement>("#pdf-edit-add-image");

  const input = root.querySelector<HTMLInputElement>("#pdf-edit-image-file");

  const width = root.querySelector<HTMLInputElement>("#pdf-edit-image-width");

  const deleteButton = root.querySelector<HTMLButtonElement>(
    "#pdf-edit-delete-image",
  );

  if (!button || !input || !width || !deleteButton) {
    return false;
  }

  button.addEventListener("click", () => {
    input.click();
  });

  input.addEventListener("change", () => {
    const file = input.files?.[0];

    input.value = "";

    if (!file) {
      return;
    }

    addImage(file);
  });

  width.addEventListener("input", () => {
    const value = Number(width.value);

    if (!Number.isFinite(value) || value < 5 || value > 100) {
      return;
    }

    updateSelectedImageWidth(value / 100);
  });

  deleteButton.addEventListener("click", deleteSelectedImage);

  return true;
}

export function renderImageObjects(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const overlay = root.querySelector<HTMLElement>("#pdf-edit-image-overlay");

  if (!overlay) {
    return;
  }

  overlay.replaceChildren();

  for (const object of getPageImageObjects(editorState.activePage)) {
    overlay.append(createImageElement(object.id));
  }
}

export function updateImageProperties(): void {
  const root = editorState.root;

  if (!root) {
    return;
  }

  const properties = root.querySelector<HTMLElement>(
    "#pdf-edit-image-properties",
  );

  const filename = root.querySelector<HTMLElement>("#pdf-edit-image-name");

  const width = root.querySelector<HTMLInputElement>("#pdf-edit-image-width");

  if (!properties || !filename || !width) {
    return;
  }

  const object = editorState.selectedImageID
    ? findImageObject(editorState.activePage, editorState.selectedImageID)
    : null;

  if (!object) {
    properties.hidden = true;

    return;
  }

  properties.hidden = false;

  filename.textContent = object.name;

  width.value = String(Math.round(object.width * 100));
}

function addImage(file: File): void {
  if (!editorState.activeUpload || !editorState.root) {
    return;
  }

  if (!allowedImageTypes.has(file.type)) {
    showImageError("Für PDF-Bilder werden aktuell PNG und JPEG unterstützt.");

    return;
  }

  if (file.size > maxImageSize) {
    showImageError("Das Bild darf maximal 32 MiB groß sein.");

    return;
  }

  const object = createImageObject(file);

  const objects = getPageImageObjects(editorState.activePage);

  objects.push(object);

  editorState.pageImageObjects.set(editorState.activePage, objects);

  editorState.selectedTextID = null;

  editorState.selectedImageID = object.id;

  hideTextProperties();
  hideImageError();

  renderImageObjects();
  updateImageProperties();
  updatePageChangeIndicators();
  updatePDFEditSubmitState();
}

function createImageElement(id: string): HTMLButtonElement {
  const object = findImageObject(editorState.activePage, id);

  const element = document.createElement("button");

  element.type = "button";

  element.className = "pdf-edit-image-object";

  if (!object) {
    return element;
  }

  element.dataset.imageId = object.id;

  element.style.left = `${object.x * 100}%`;

  element.style.top = `${object.y * 100}%`;

  element.style.width = `${object.width * 100}%`;

  element.classList.toggle(
    "selected",
    object.id === editorState.selectedImageID,
  );

  const image = document.createElement("img");

  image.src = object.url;

  image.alt = object.name;

  image.draggable = false;

  element.append(image);

  element.addEventListener("click", (event) => {
    event.stopPropagation();

    selectImageObject(object.id);
  });

  element.addEventListener("pointerdown", (event) => {
    beginImageDrag(event, object.id, element);
  });

  element.addEventListener("pointermove", moveImageDrag);

  element.addEventListener("pointerup", endImageDrag);

  element.addEventListener("pointercancel", endImageDrag);

  return element;
}

function selectImageObject(id: string): void {
  editorState.selectedTextID = null;

  editorState.selectedImageID = id;

  hideTextProperties();

  renderImageObjects();
  updateImageProperties();
}

function beginImageDrag(
  event: PointerEvent,
  id: string,
  element: HTMLElement,
): void {
  event.preventDefault();

  const rectangle = element.getBoundingClientRect();

  editorState.selectedTextID = null;

  editorState.selectedImageID = id;

  hideTextProperties();

  editorState.dragState = {
    id,

    pointerId: event.pointerId,

    offsetX: event.clientX - rectangle.left,

    offsetY: event.clientY - rectangle.top,
  };

  element.setPointerCapture(event.pointerId);

  updateImageProperties();
}

function moveImageDrag(event: PointerEvent): void {
  const root = editorState.root;

  const drag = editorState.dragState;

  if (!root || !drag || drag.pointerId !== event.pointerId) {
    return;
  }

  const element = event.currentTarget;

  if (!(element instanceof HTMLElement)) {
    return;
  }

  const overlay = root.querySelector<HTMLElement>("#pdf-edit-image-overlay");

  if (!overlay) {
    return;
  }

  const rectangle = overlay.getBoundingClientRect();

  if (rectangle.width <= 0 || rectangle.height <= 0) {
    return;
  }

  const x = (event.clientX - rectangle.left - drag.offsetX) / rectangle.width;

  const y = (event.clientY - rectangle.top - drag.offsetY) / rectangle.height;

  setImagePosition(drag.id, x, y, element);
}

function endImageDrag(event: PointerEvent): void {
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

function setImagePosition(
  id: string,
  x: number,
  y: number,
  element: HTMLElement,
): void {
  const object = findImageObject(editorState.activePage, id);

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

function updateSelectedImageWidth(width: number): void {
  const id = editorState.selectedImageID;

  if (!id) {
    return;
  }

  const object = findImageObject(editorState.activePage, id);

  if (!object) {
    return;
  }

  object.width = clamp(width, 0.05, 1);

  if (object.x + object.width > 1) {
    object.x = Math.max(0, 1 - object.width);
  }

  renderImageObjects();
  updateImageProperties();
  updatePDFEditSubmitState();
}

function deleteSelectedImage(): void {
  const id = editorState.selectedImageID;

  if (!id) {
    return;
  }

  const object = findImageObject(editorState.activePage, id);

  if (object) {
    URL.revokeObjectURL(object.url);
  }

  const objects = getPageImageObjects(editorState.activePage).filter(
    (candidate) => candidate.id !== id,
  );

  editorState.pageImageObjects.set(editorState.activePage, objects);

  editorState.selectedImageID = null;

  renderImageObjects();
  updateImageProperties();
  updatePageChangeIndicators();
  updatePDFEditSubmitState();
}

function hideTextProperties(): void {
  const properties = editorState.root?.querySelector<HTMLElement>(
    "#pdf-edit-text-properties",
  );

  if (properties) {
    properties.hidden = true;
  }
}

function showImageError(message: string): void {
  const element =
    editorState.root?.querySelector<HTMLElement>("#pdf-edit-error");

  if (!element) {
    return;
  }

  element.textContent = message;

  element.hidden = false;
}

function hideImageError(): void {
  const element =
    editorState.root?.querySelector<HTMLElement>("#pdf-edit-error");

  if (!element) {
    return;
  }

  element.textContent = "";

  element.hidden = true;
}
