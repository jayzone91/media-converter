import { downloadBlob } from "../shared/download.ts";

const PNG_SIZE = 1200;

export function setupQRDownloads(
  getSVG: () => string,
  showStatus: (text: string) => void,
): void {
  const container = document.querySelector<HTMLElement>(".qr-download-actions");

  if (!container) {
    return;
  }

  const buttons = container.querySelectorAll<HTMLButtonElement>("button");

  const pngButton = buttons.item(0);

  const svgButton = buttons.item(1);

  if (pngButton) {
    pngButton.textContent = "PNG herunterladen";

    pngButton.addEventListener("click", () => {
      void downloadQRPNG(getSVG(), showStatus);
    });
  }

  if (svgButton) {
    svgButton.textContent = "SVG herunterladen";

    svgButton.addEventListener("click", () => {
      downloadQRSVG(getSVG());
    });
  }
}

function downloadQRSVG(svg: string): void {
  if (!svg) {
    return;
  }

  const blob = new Blob([svg], {
    type: "image/svg+xml;charset=utf-8",
  });

  downloadBlob(blob, "qr-code.svg");
}

async function downloadQRPNG(
  svg: string,
  showStatus: (text: string) => void,
): Promise<void> {
  if (!svg) {
    return;
  }

  try {
    const svgBlob = new Blob([svg], {
      type: "image/svg+xml;charset=utf-8",
    });

    const svgURL = URL.createObjectURL(svgBlob);

    try {
      const image = await loadImage(svgURL);

      const canvas = document.createElement("canvas");

      canvas.width = PNG_SIZE;

      canvas.height = PNG_SIZE;

      const context = canvas.getContext("2d");

      if (!context) {
        throw new Error("Canvas konnte nicht erstellt werden.");
      }

      context.drawImage(image, 0, 0, PNG_SIZE, PNG_SIZE);

      const png = await canvasToBlob(canvas);

      downloadBlob(png, "qr-code.png");
    } finally {
      URL.revokeObjectURL(svgURL);
    }
  } catch (error: unknown) {
    console.error("PNG Export fehlgeschlagen:", error);

    showStatus("Exportfehler");
  }
}

function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();

    image.addEventListener("load", () => {
      resolve(image);
    });

    image.addEventListener("error", () => {
      reject(new Error("SVG konnte nicht für PNG gerendert werden."));
    });

    image.src = url;
  });
}

function canvasToBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        reject(new Error("PNG konnte nicht erzeugt werden."));

        return;
      }

      resolve(blob);
    }, "image/png");
  });
}
