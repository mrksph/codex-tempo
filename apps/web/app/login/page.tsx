import { LockKeyhole } from "lucide-react";
import { redirect } from "next/navigation";
import { hasSession } from "@/lib/auth/session";
import { login } from "./actions";

export const dynamic = "force-dynamic";

export default async function Login({ searchParams }: { searchParams: Promise<{ error?: string }> }) {
  if (await hasSession()) redirect("/dashboard"); const { error } = await searchParams;
  return <main className="grid min-h-screen place-items-center p-6"><form className="w-[min(360px,100%)] border border-[var(--line)] bg-white p-6" action={login}><LockKeyhole size={22}/><h1 className="m-0 mb-1.5 text-[20px]">Codex Tempo</h1><p className="mb-5 text-sm text-[var(--muted)]">Private dashboard access.</p>{error && <p className="text-[var(--danger)]">The password is not valid.</p>}<input name="username" type="text" autoComplete="username" value="tempo" readOnly hidden/><label className="grid gap-1.5 text-sm font-medium"><span>Password</span><input className="h-10 border border-[var(--line)] px-2.5" name="password" type="password" autoComplete="current-password" required autoFocus/></label><button className="inline-flex w-full min-h-9 items-center justify-center gap-2 rounded border-0 bg-[var(--accent)] px-3 text-xs font-bold text-white" type="submit">Sign in</button></form></main>;
}
