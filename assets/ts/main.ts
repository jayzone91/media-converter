import "htmx.org";

import { setupConverter } from "./converter/index.ts";

import { setupPDFUI } from "./pdf/index.ts";

import { setupQRUI } from "./qr/index.ts";

function setupTabs(): void {
  const buttons = document.querySelectorAll<HTMLButtonElement>(".tab-button");

  const panels = document.querySelectorAll<HTMLElement>(".tab-panel");

  for (const button of buttons) {
    button.addEventListener("click", () => {
      const target = button.dataset.tab;

      if (!target) {
        return;
      }

      for (const candidate of buttons) {
        candidate.classList.toggle("active", candidate === button);
      }

      for (const panel of panels) {
        const active = panel.dataset.panel === target;

        panel.classList.toggle("active", active);

        panel.hidden = !active;
      }
    });
  }
}

document.addEventListener("DOMContentLoaded", () => {
  setupTabs();
  setupConverter();
  setupPDFUI();
  setupQRUI();
});
