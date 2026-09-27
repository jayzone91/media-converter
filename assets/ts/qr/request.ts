import type { QRRequest, QRType } from "./types.ts";

let qrLogoDataURL = "";

export function setQRLogoDataURL(value: string): void {
  qrLogoDataURL = value;
}

export function getQRLogoDataURL(): string {
  return qrLogoDataURL;
}

export function getActiveQRType(): QRType {
  const button = document.querySelector<HTMLElement>(".qr-type.active");

  return parseQRType(button?.dataset.qrType);
}

export function buildQRRequest(): QRRequest {
  const colorMode = getCheckedInputValue(
    'input[name="qr_color_mode"]:checked',
    "solid",
  );

  return {
    type: getActiveQRType(),

    url: getInputValue("qr_url"),

    text: getInputValue("qr_text"),

    phone: getInputValue("qr_phone"),

    wifi: {
      ssid: getInputValue("wifi_ssid"),

      password: getInputValue("wifi_password"),

      encryption: getInputValue("wifi_encryption"),

      hidden: getCheckboxValue("wifi_hidden"),
    },

    vcard: {
      first_name: getInputValue("vcard_firstname"),

      last_name: getInputValue("vcard_lastname"),

      company: getInputValue("vcard_company"),

      position: getInputValue("vcard_position"),

      phone_work: getInputValue("vcard_phone_work"),

      phone_home: getInputValue("vcard_phone_home"),

      mobile_work: getInputValue("vcard_mobile_work"),

      mobile_home: getInputValue("vcard_mobile_home"),

      fax_work: getInputValue("vcard_fax_work"),

      email: getInputValue("vcard_email"),

      website: getInputValue("vcard_website"),

      street: getInputValue("vcard_street"),

      postal_code: getInputValue("vcard_postal_code"),

      city: getInputValue("vcard_city"),

      region: getInputValue("vcard_region"),

      country: getInputValue("vcard_country"),
    },

    event: {
      title: getInputValue("event_title"),

      start: getInputValue("event_start"),

      end: getInputValue("event_end"),

      location: getInputValue("event_location"),

      description: getInputValue("event_description"),
    },

    style: {
      foreground: getElementValue("#qr-foreground", "#000000"),

      background: getElementValue("#qr-background", "#ffffff"),

      gradient_enabled: colorMode === "gradient",

      gradient_start: getElementValue("#qr-gradient-start", "#000000"),

      gradient_end: getElementValue("#qr-gradient-end", "#3b82f6"),

      module: getActiveQRStyle("dots"),

      corner_outer: getActiveQRStyle("corners-square"),

      corner_inner: getActiveQRStyle("corners-dot"),

      has_logo: qrLogoDataURL !== "",

      logo: qrLogoDataURL,
    },
  };
}

export function hasQRPayload(request: QRRequest): boolean {
  switch (request.type) {
    case "url":
      return request.url.trim() !== "";

    case "text":
      return request.text.trim() !== "";

    case "phone":
      return request.phone.trim() !== "";

    case "wifi":
      return request.wifi.ssid.trim() !== "";

    case "vcard":
      return (
        request.vcard.first_name.trim() !== "" ||
        request.vcard.last_name.trim() !== ""
      );

    case "event":
      return (
        request.event.title.trim() !== "" &&
        request.event.start !== "" &&
        request.event.end !== ""
      );
  }
}

function parseQRType(value: string | undefined): QRType {
  switch (value) {
    case "text":
    case "phone":
    case "wifi":
    case "vcard":
    case "event":
      return value;

    case "url":
    default:
      return "url";
  }
}

function getActiveQRStyle(group: string): string {
  const element = document.querySelector<HTMLElement>(
    `[data-style-group="${group}"] .style-option.active`,
  );

  return element?.dataset.style ?? "square";
}

function getInputValue(name: string): string {
  const element = document.querySelector<
    HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement
  >(`[name="${name}"]`);

  return element?.value ?? "";
}

function getCheckboxValue(name: string): boolean {
  const element = document.querySelector<HTMLInputElement>(`[name="${name}"]`);

  return element?.checked ?? false;
}

function getCheckedInputValue(selector: string, fallback: string): string {
  const element = document.querySelector<HTMLInputElement>(selector);

  return element?.value ?? fallback;
}

function getElementValue(selector: string, fallback: string): string {
  const element = document.querySelector<HTMLInputElement>(selector);

  return element?.value ?? fallback;
}
