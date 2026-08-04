"use server";

import { redirect } from "next/navigation";
import { createSession } from "@/lib/auth/session";

export async function login(formData: FormData) {
  if (String(formData.get("password") || "") !== (process.env.WEB_PASSWORD || "")) redirect("/login?error=1");
  await createSession(); redirect("/dashboard");
}
