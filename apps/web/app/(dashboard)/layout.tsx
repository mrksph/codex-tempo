import { redirect } from "next/navigation";
import { Sidebar } from "@/components/sidebar";
import { QueryProvider } from "@/components/query-provider";
import { hasSession } from "@/lib/auth/session";

export default async function DashboardLayout({ children }: { children: React.ReactNode }) {
  if (!(await hasSession())) redirect("/login");
  return <QueryProvider><div className="min-h-screen grid grid-cols-[224px_minmax(0,1fr)] max-[760px]:block"><Sidebar/><div className="min-w-0"><header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b border-[var(--line)] bg-[#ffffffde] px-7 backdrop-blur-sm max-[760px]:h-[54px] max-[760px]:px-4"><span className="text-sm font-bold">Agent activity</span><span className="flex items-center gap-1.5 text-xs text-[var(--muted)]"><span className="h-2 w-2 rounded-full bg-[#20a474] shadow-[0_0_0_3px_#dcf4eb]"/>System connected</span></header><main className="mx-auto w-full max-w-[1500px] px-7 pb-12 pt-7 max-[760px]:px-4">{children}</main></div></div></QueryProvider>;
}
