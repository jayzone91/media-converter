import type { DownloadResult } from "../../shared/download.ts";

export interface PDFUploadResponse {
  id: string;
  filename: string;
  size: number;
  page_count: number;
  previews: string[];
}

export interface MergeDocument {
  id: string;
  filename: string;
  size: number;
  pageCount: number;
  previews: string[];
}

export type MergeResult = DownloadResult;

export interface DocumentRenderActions {
  moveUp: (id: string) => void;
  moveDown: (id: string) => void;
  remove: (id: string) => void;
  reorder: (sourceID: string, targetID: string) => void;
}
