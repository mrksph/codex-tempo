import { LockKeyhole } from "lucide-react";
import { redirect } from "next/navigation";
import { hasSession } from "@/lib/auth/session";
import { login } from "./actions";

export const dynamic = "force-dynamic";

export default async function Login({ searchParams }: { searchParams: Promise<{ error?: string }> }) {
  if (await hasSession()) redirect("/dashboard"); const { error } = await searchParams;
  return <main className="login"><form className="login-box" action={login}><LockKeyhole size={22}/><h1>Codex Tempo</h1><p>Private dashboard access.</p>{error && <p style={{color:"var(--danger)"}}>The password is not valid.</p>}<input name="username" type="text" autoComplete="username" value="tempo" readOnly hidden/><label className="field">Password<input name="password" type="password" autoComplete="current-password" required autoFocus/></label><button className="button" type="submit">Sign in</button></form></main>;
}
