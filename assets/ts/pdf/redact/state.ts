import type { PDFUpload } from "../uploads.ts";

export interface PDFRedaction {
  id: string;

  x: number;
  y: number;

  width: number;
  height: number;
}

interface PDFRedactState {
  root: HTMLElement | null;

  activeUpload: PDFUpload | null;

  activePage: number;

  selectedID: string | null;

  sequence: number;

  pageRedactions: Map<number, PDFRedaction[]>;
}

export const redactState: PDFRedactState = {
  root: null,

  activeUpload: null,

  activePage: 0,

  selectedID: null,

  sequence: 0,

  pageRedactions: new Map<number, PDFRedaction[]>(),
};

export function clearRedactState(): void {
  redactState.pageRedactions.clear();

  redactState.selectedID = null;

  redactState.sequence = 0;
}

export function getPageRedactions(page: number): PDFRedaction[] {
  return redactState.pageRedactions.get(page) ?? [];
}

export function findRedaction(page: number, id: string): PDFRedaction | null {
  return (
    getPageRedactions(page).find((redaction) => redaction.id === id) ?? null
  );
}

export function createRedaction(
  x: number,
  y: number,
  width: number,
  height: number,
): PDFRedaction {
  redactState.sequence++;

  return {
    id: `redaction-${redactState.sequence}`,

    x,
    y,

    width,
    height,
  };
}

export function pageHasRedactions(page: number): boolean {
  return getPageRedactions(page).length > 0;
}

export function hasRedactions(): boolean {
  for (const redactions of redactState.pageRedactions.values()) {
    if (redactions.length > 0) {
      return true;
    }
  }

  return false;
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
