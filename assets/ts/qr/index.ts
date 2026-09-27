import { setupQRLogo } from "./logo.ts";

import { setupQRDownloads } from "./download.ts";

import {
  generateQRPreview,
  getLastGeneratedQRSVG,
  scheduleQRPreview,
  showQRStatus,
} from "./preview.ts";

export function setupQRUI(): void {
  setupQRTypes();
  setupQRStyles();
  setupQRColorMode();
  setupQRLivePreview();
  setupQRColorLabels();
  setupQRFieldSwaps();

  setupQRLogo(scheduleQRPreview, showQRStatus);

  setupQRDownloads(getLastGeneratedQRSVG, showQRStatus);
}

function setupQRTypes(): void {
  const buttons = document.querySelectorAll<HTMLElement>(".qr-type");

  for (const button of buttons) {
    button.addEventListener("click", () => {
      for (const candidate of buttons) {
        candidate.classList.toggle("active", candidate === button);
      }
    });
  }
}

function setupQRFieldSwaps(): void {
  const fields = document.querySelector<HTMLElement>("#qr-fields");

  if (!fields) {
    return;
  }

  fields.addEventListener("htmx:afterSwap", () => {
    scheduleQRPreview();
  });
}

function setupQRStyles(): void {
  const groups = document.querySelectorAll<HTMLElement>("[data-style-group]");

  for (const group of groups) {
    const options = group.querySelectorAll<HTMLElement>(".style-option");

    for (const option of options) {
      option.addEventListener("click", () => {
        for (const candidate of options) {
          candidate.classList.toggle("active", candidate === option);
        }

        scheduleQRPreview();
      });
    }
  }
}

function setupQRColorMode(): void {
  const colorModes = document.querySelectorAll<HTMLInputElement>(
    'input[name="qr_color_mode"]',
  );

  const gradientSettings = document.querySelector<HTMLElement>(
    "#qr-gradient-settings",
  );

  if (!gradientSettings) {
    return;
  }

  for (const mode of colorModes) {
    mode.addEventListener("change", () => {
      const selected = document.querySelector<HTMLInputElement>(
        'input[name="qr_color_mode"]:checked',
      );

      gradientSettings.hidden = selected?.value !== "gradient";

      scheduleQRPreview();
    });
  }
}

function setupQRLivePreview(): void {
  const panel = document.querySelector<HTMLElement>('[data-panel="qr"]');

  if (!panel) {
    return;
  }

  const handleInput = (event: Event): void => {
    const target = event.target;

    if (
      target instanceof HTMLInputElement ||
      target instanceof HTMLTextAreaElement ||
      target instanceof HTMLSelectElement
    ) {
      scheduleQRPreview();
    }
  };

  panel.addEventListener("input", handleInput);

  panel.addEventListener("change", handleInput);

  const generateButton =
    document.querySelector<HTMLButtonElement>("#qr-generate");

  generateButton?.addEventListener("click", () => {
    void generateQRPreview();
  });
}

function setupQRColorLabels(): void {
  const ids = [
    "qr-foreground",
    "qr-background",
    "qr-gradient-start",
    "qr-gradient-end",
  ] as const;

  for (const id of ids) {
    const input = document.querySelector<HTMLInputElement>(`#${id}`);

    if (!input) {
      continue;
    }

    updateQRColorLabel(input);

    input.addEventListener("input", () => {
      updateQRColorLabel(input);
    });
  }
}

function updateQRColorLabel(input: HTMLInputElement): void {
  const code = input.parentElement?.querySelector<HTMLElement>("code");

  if (code) {
    code.textContent = input.value.toUpperCase();
  }
}
