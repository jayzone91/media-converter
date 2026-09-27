export type QRType = "url" | "text" | "phone" | "wifi" | "vcard" | "event";

export interface QRWiFi {
  ssid: string;
  password: string;
  encryption: string;
  hidden: boolean;
}

export interface QRVCard {
  first_name: string;
  last_name: string;
  company: string;
  position: string;
  phone_work: string;
  phone_home: string;
  mobile_work: string;
  mobile_home: string;
  fax_work: string;
  email: string;
  website: string;
  street: string;
  postal_code: string;
  city: string;
  region: string;
  country: string;
}

export interface QREvent {
  title: string;
  start: string;
  end: string;
  location: string;
  description: string;
}

export interface QRStyleRequest {
  foreground: string;
  background: string;
  gradient_enabled: boolean;
  gradient_start: string;
  gradient_end: string;
  module: string;
  corner_outer: string;
  corner_inner: string;
  has_logo: boolean;
  logo: string;
}

export interface QRRequest {
  type: QRType;

  url: string;
  text: string;
  phone: string;

  wifi: QRWiFi;
  vcard: QRVCard;
  event: QREvent;

  style: QRStyleRequest;
}

export interface QRResponse {
  svg: string;
  version: number;
  error_correction: string;
}
