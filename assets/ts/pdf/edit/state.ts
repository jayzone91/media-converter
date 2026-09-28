import type { PDFUpload } from "../uploads.ts";

export interface PDFTextObject {
  id: string;
  text: string;
  x: number;
  y: number;
  size: number;
  color: string;
}

export interface PDFImageObject {
  id: string;
  name: string;
  file: File;
  url: string;
  x: number;
  y: number;
  width: number;
}

export interface PDFEditDragState {
  id: string;
  pointerId: number;
  offsetX: number;
  offsetY: number;
}

interface PDFEditState {
  activeUpload: PDFUpload | null;
  activePage: number;
  root: HTMLElement | null;

  selectedTextID: string | null;
  selectedImageID: string | null;

  dragState: PDFEditDragState | null;

  textSequence: number;
  imageSequence: number;

  pageTextObjects: Map<number, PDFTextObject[]>;
  pageImageObjects: Map<number, PDFImageObject[]>;
}

export const editorState: PDFEditState = {
  activeUpload: null,
  activePage: 0,
  root: null,

  selectedTextID: null,
  selectedImageID: null,

  dragState: null,

  textSequence: 0,
  imageSequence: 0,

  pageTextObjects: new Map<number, PDFTextObject[]>(),
  pageImageObjects: new Map<number, PDFImageObject[]>(),
};

export function clearEditorState(): void {
  revokeImageObjectURLs();

  editorState.pageTextObjects.clear();
  editorState.pageImageObjects.clear();

  editorState.selectedTextID = null;
  editorState.selectedImageID = null;
  editorState.dragState = null;

  editorState.textSequence = 0;
  editorState.imageSequence = 0;

  const textProperties = editorState.root?.querySelector<HTMLElement>(
    "#pdf-edit-text-properties",
  );

  const imageProperties = editorState.root?.querySelector<HTMLElement>(
    "#pdf-edit-image-properties",
  );

  if (textProperties) {
    textProperties.hidden = true;
  }

  if (imageProperties) {
    imageProperties.hidden = true;
  }
}

export function getPageTextObjects(page: number): PDFTextObject[] {
  return editorState.pageTextObjects.get(page) ?? [];
}

export function getPageImageObjects(page: number): PDFImageObject[] {
  return editorState.pageImageObjects.get(page) ?? [];
}

export function findTextObject(page: number, id: string): PDFTextObject | null {
  return getPageTextObjects(page).find((object) => object.id === id) ?? null;
}

export function findImageObject(
  page: number,
  id: string,
): PDFImageObject | null {
  return getPageImageObjects(page).find((object) => object.id === id) ?? null;
}

export function createTextObject(): PDFTextObject {
  editorState.textSequence++;

  return {
    id: `text-${editorState.textSequence}`,
    text: "Text",
    x: 0.1,
    y: 0.1,
    size: 18,
    color: "#000000",
  };
}

export function createImageObject(file: File): PDFImageObject {
  editorState.imageSequence++;

  return {
    id: `image-${editorState.imageSequence}`,
    name: file.name,
    file,
    url: URL.createObjectURL(file),
    x: 0.1,
    y: 0.1,
    width: 0.25,
  };
}

export function pageHasEdits(page: number): boolean {
  return (
    getPageTextObjects(page).length > 0 || getPageImageObjects(page).length > 0
  );
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function revokeImageObjectURLs(): void {
  for (const objects of editorState.pageImageObjects.values()) {
    for (const object of objects) {
      URL.revokeObjectURL(object.url);
    }
  }
}
