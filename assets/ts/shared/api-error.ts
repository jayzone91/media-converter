interface APIErrorResponse {
  code?: unknown;
  message?: unknown;
}

export interface APIError {
  code: string | null;
  message: string;
}

export async function readAPIError(
  response: Response,
  fallback: string,
): Promise<APIError> {
  const contentType = response.headers.get("Content-Type") ?? "";

  if (contentType.includes("application/json")) {
    try {
      const payload = (await response.json()) as APIErrorResponse;

      const code = typeof payload.code === "string" ? payload.code : null;

      const message =
        typeof payload.message === "string" && payload.message.trim() !== ""
          ? payload.message.trim()
          : fallback;

      return {
        code,
        message,
      };
    } catch {
      return {
        code: null,
        message: fallback,
      };
    }
  }

  try {
    const message = (await response.text()).trim();

    return {
      code: null,
      message: message || fallback,
    };
  } catch {
    return {
      code: null,
      message: fallback,
    };
  }
}
