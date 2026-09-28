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

export interface PDFDrawPoint {
  x: number;
  y: number;
}

export interface PDFDrawStroke {
  id: string;
  color: string;
  width: number;
  points: PDFDrawPoint[];
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
  drawSequence: number;

  pageTextObjects: Map<number, PDFTextObject[]>;
  pageImageObjects: Map<number, PDFImageObject[]>;
  pageDrawStrokes: Map<number, PDFDrawStroke[]>;
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
  drawSequence: 0,

  pageTextObjects: new Map<number, PDFTextObject[]>(),
  pageImageObjects: new Map<number, PDFImageObject[]>(),
  pageDrawStrokes: new Map<number, PDFDrawStroke[]>(),
};

export function clearEditorState(): void {
  revokeImageObjectURLs();

  editorState.pageTextObjects.clear();
  editorState.pageImageObjects.clear();
  editorState.pageDrawStrokes.clear();

  editorState.selectedTextID = null;
  editorState.selectedImageID = null;
  editorState.dragState = null;

  editorState.textSequence = 0;
  editorState.imageSequence = 0;
  editorState.drawSequence = 0;

  const textProperties = editorState.root?.querySelector<HTMLElement>(
    "#pdf-edit-text-properties",
  );

  const imageProperties = editorState.root?.querySelector<HTMLElement>(
    "#pdf-edit-image-properties",
  );

  const drawProperties = editorState.root?.querySelector<HTMLElement>(
    "#pdf-edit-draw-properties",
  );

  if (textProperties) {
    textProperties.hidden = true;
  }

  if (imageProperties) {
    imageProperties.hidden = true;
  }

  if (drawProperties) {
    drawProperties.hidden = true;
  }
}

export function getPageTextObjects(page: number): PDFTextObject[] {
  return editorState.pageTextObjects.get(page) ?? [];
}

export function getPageImageObjects(page: number): PDFImageObject[] {
  return editorState.pageImageObjects.get(page) ?? [];
}

export function getPageDrawStrokes(page: number): PDFDrawStroke[] {
  return editorState.pageDrawStrokes.get(page) ?? [];
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

export function createDrawStroke(
  color: string,
  width: number,
  point: PDFDrawPoint,
): PDFDrawStroke {
  editorState.drawSequence++;

  return {
    id: `draw-${editorState.drawSequence}`,
    color,
    width,
    points: [point],
  };
}

export function pageHasEdits(page: number): boolean {
  return (
    getPageTextObjects(page).length > 0 ||
    getPageImageObjects(page).length > 0 ||
    getPageDrawStrokes(page).length > 0
  );
}

export function hasDrawStrokes(): boolean {
  for (const strokes of editorState.pageDrawStrokes.values()) {
    if (strokes.length > 0) {
      return true;
    }
  }

  return false;
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
