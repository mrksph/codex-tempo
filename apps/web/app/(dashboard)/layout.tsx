import { redirect } from "next/navigation";
import { Sidebar } from "@/components/sidebar";
import { QueryProvider } from "@/components/query-provider";
import { hasSession } from "@/lib/auth/session";

export default async function DashboardLayout({ children }: { children: React.ReactNode }) {
  if (!(await hasSession())) redirect("/login");
  return <QueryProvider><div className="shell"><Sidebar/><div className="main"><header className="topbar"><span className="topbar-title">Actividad de agentes</span><span className="status"><span className="status-dot"/>Sistema conectado</span></header><main className="content">{children}</main></div></div></QueryProvider>;
}
