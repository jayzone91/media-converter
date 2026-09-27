import type { PDFUpload } from "../uploads.ts";

export interface PDFTextObject {
  id: string;
  text: string;
  x: number;
  y: number;
  size: number;
  color: string;
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
  dragState: PDFEditDragState | null;
  textSequence: number;
  pageTextObjects: Map<number, PDFTextObject[]>;
}

export const editorState: PDFEditState = {
  activeUpload: null,
  activePage: 0,
  root: null,
  selectedTextID: null,
  dragState: null,
  textSequence: 0,
  pageTextObjects: new Map<number, PDFTextObject[]>(),
};

export function clearEditorState(): void {
  editorState.pageTextObjects.clear();

  editorState.selectedTextID = null;
  editorState.dragState = null;
  editorState.textSequence = 0;

  const properties = editorState.root?.querySelector<HTMLElement>(
    "#pdf-edit-text-properties",
  );

  if (properties) {
    properties.hidden = true;
  }
}

export function getPageTextObjects(page: number): PDFTextObject[] {
  return editorState.pageTextObjects.get(page) ?? [];
}

export function findTextObject(page: number, id: string): PDFTextObject | null {
  return getPageTextObjects(page).find((object) => object.id === id) ?? null;
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

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
