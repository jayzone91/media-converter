import { requestDownload, type DownloadResult } from "../../shared/download.ts";

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

export type PDFWebResult = DownloadResult;

export async function createWebPDF(
  options: PDFWebOptions,
): Promise<PDFWebResult> {
  return requestDownload(
    "/pdf/web",
    {
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
    },
    "Die Webseite konnte nicht als PDF erstellt werden.",
  );
}
