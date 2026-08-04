import "server-only";

import { createHmac, timingSafeEqual } from "node:crypto";
import { cookies } from "next/headers";

const COOKIE_NAME = "tempo_session";

function signature(value: string) {
  return createHmac("sha256", process.env.AUTH_SECRET || "development-only-secret")
    .update(value)
    .digest("base64url");
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
  (await cookies()).set(COOKIE_NAME, `${expires}.${signature(expires)}`, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 12 * 60 * 60,
  });
}

export async function clearSession() {
  (await cookies()).delete(COOKIE_NAME);
}
