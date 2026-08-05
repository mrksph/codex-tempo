import "server-only";

import { createHmac, timingSafeEqual } from "node:crypto";
import { cookies, headers } from "next/headers";

const COOKIE_NAME = "tempo_session";

function signature(value: string) {
  return createHmac("sha256", process.env.AUTH_SECRET || "development-only-secret")
    .update(value)
    .digest("base64url");
}

async function secureCookieEnabled() {
  const forced = (process.env.AUTH_COOKIE_SECURE || "").toLowerCase();
  if (forced === "true") return true;
  if (forced === "false") return false;
  const requestHeaders = await headers();
  const proto = (requestHeaders.get("x-forwarded-proto") || requestHeaders.get("x-url-scheme") || "").split(",")[0].trim().toLowerCase();
  return proto === "https";
}

export async function hasSession() {
  if (!process.env.WEB_PASSWORD) return true;
  const raw = (await cookies()).get(COOKIE_NAME)?.value;
  if (!raw) return false;
  const [expires, supplied] = raw.split(".");
  if (!expires || !supplied || Number(expires) < Date.now()) return false;
  const expected = signature(expires);
  return supplied.length === expected.length && timingSafeEqual(Buffer.from(supplied), Buffer.from(expected));
}

export async function createSession() {
  const expires = String(Date.now() + 12 * 60 * 60 * 1000);
  const secure = await secureCookieEnabled();
  (await cookies()).set(COOKIE_NAME, `${expires}.${signature(expires)}`, {
    httpOnly: true,
    sameSite: "lax",
    secure,
    path: "/",
    maxAge: 12 * 60 * 60,
  });
}

export async function clearSession() {
  (await cookies()).delete(COOKIE_NAME);
}
