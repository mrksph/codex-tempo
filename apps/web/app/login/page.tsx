import { LockKeyhole } from "lucide-react";
import { redirect } from "next/navigation";
import { hasSession } from "@/lib/auth/session";
import { login } from "./actions";

export const dynamic = "force-dynamic";

export default async function Login({ searchParams }: { searchParams: Promise<{ error?: string }> }) {
  if (await hasSession()) redirect("/dashboard"); const { error } = await searchParams;
  return <main className="login"><form className="login-box" action={login}><LockKeyhole size={22}/><h1>Codex Tempo</h1><p>Acceso al panel privado.</p>{error && <p style={{color:"var(--danger)"}}>La contraseña no es válida.</p>}<input name="username" type="text" autoComplete="username" value="tempo" readOnly hidden/><label className="field">Contraseña<input name="password" type="password" autoComplete="current-password" required autoFocus/></label><button className="button" type="submit">Entrar</button></form></main>;
}
