import { setQRLogoDataURL } from "./request.ts";

export function setupQRLogo(
  schedulePreview: () => void,
  showStatus: (text: string) => void,
): void {
  const input = document.querySelector<HTMLInputElement>("#qr-logo");

  if (!input) {
    return;
  }

  input.accept = "image/png,image/jpeg,image/webp";

  const label = input.closest<HTMLElement>(".logo-upload");

  const title = label?.querySelector<HTMLElement>("strong");

  const subtitle = label?.querySelector<HTMLElement>("small");

  if (subtitle) {
    subtitle.textContent = "PNG, JPEG oder WEBP · max. 2 MiB";
  }

  input.addEventListener("change", () => {
    void handleLogoChange(input, title ?? null, schedulePreview, showStatus);
  });
}

async function handleLogoChange(
  input: HTMLInputElement,
  title: HTMLElement | null,
  schedulePreview: () => void,
  showStatus: (text: string) => void,
): Promise<void> {
  const file = input.files?.item(0);

  if (!file) {
    setQRLogoDataURL("");

    if (title) {
      title.textContent = "Logo auswählen";
    }

    schedulePreview();

    return;
  }

  const allowedTypes = new Set(["image/png", "image/jpeg", "image/webp"]);

  if (!allowedTypes.has(file.type)) {
    input.value = "";

    setQRLogoDataURL("");

    if (title) {
      title.textContent = "Ungültiges Dateiformat";
    }

    showStatus("Fehler");

    return;
  }

  if (file.size > 2 * 1024 * 1024) {
    input.value = "";

    setQRLogoDataURL("");

    if (title) {
      title.textContent = "Logo ist größer als 2 MiB";
    }

    showStatus("Fehler");

    return;
  }

  try {
    const result = await readFileAsDataURL(file);

    setQRLogoDataURL(result);

    if (title) {
      title.textContent = file.name;
    }

    schedulePreview();
  } catch (error: unknown) {
    console.error("Logo konnte nicht gelesen werden:", error);

    input.value = "";

    setQRLogoDataURL("");

    if (title) {
      title.textContent = "Logo konnte nicht gelesen werden";
    }

    showStatus("Fehler");
  }
}

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();

    reader.addEventListener("load", () => {
      if (typeof reader.result === "string") {
        resolve(reader.result);

        return;
      }

      reject(new Error("Logo konnte nicht gelesen werden."));
    });

    reader.addEventListener("error", () => {
      reject(reader.error ?? new Error("Logo konnte nicht gelesen werden."));
    });

    reader.readAsDataURL(file);
  });
}
