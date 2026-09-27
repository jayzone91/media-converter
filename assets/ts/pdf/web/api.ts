import { getDownloadFilename } from "../../shared/download.ts";

export type PDFWebPaperSize = "a4" | "letter";

export type PDFWebRenderMode = "desktop" | "tablet" | "mobile" | "print";

export interface PDFWebOptions {
  url: string;

  paperSize: PDFWebPaperSize;

  renderMode: PDFWebRenderMode;

  landscape: boolean;

  printBackground: boolean;

  waitMilliseconds: number;
}

export interface PDFWebResult {
  blob: Blob;
  filename: string;
}

export async function createWebPDF(
  options: PDFWebOptions,
): Promise<PDFWebResult> {
  const response = await fetch("/pdf/web", {
    method: "POST",

    headers: {
      "Content-Type": "application/json",
    },

    body: JSON.stringify({
      url: options.url,

      paper_size: options.paperSize,

      render_mode: options.renderMode,

      landscape: options.landscape,

      print_background: options.printBackground,

      wait_ms: options.waitMilliseconds,
    }),
  });

  if (!response.ok) {
    const message = await response.text();

    throw new Error(
      message.trim() || "Die Webseite konnte nicht als PDF erstellt werden.",
    );
  }

  return {
    blob: await response.blob(),

    filename: getDownloadFilename(
      response.headers.get("Content-Disposition"),
      "webseite.pdf",
    ),
  };
}
