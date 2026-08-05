import { Clock3, Cpu, FolderKanban, LayoutDashboard, Settings, TerminalSquare } from "lucide-react";
import Link from "next/link";

const links = [
  ["/dashboard", "Summary", LayoutDashboard],
  ["/projects", "Projects", FolderKanban],
  ["/sessions", "Sessions", TerminalSquare],
  ["/machines", "Machines", Cpu],
  ["/settings", "Settings", Settings],
] as const;

export function Sidebar() {
  return <aside className="sticky top-0 min-h-screen flex-col bg-[#17201d] px-[14px] py-[22px] text-[#dbe4e0] flex max-[760px]:static max-[760px]:h-auto max-[760px]:w-full max-[760px]:p-3">
    <Link href="/dashboard" className="mb-[14px] flex items-center gap-2.5 px-2.5 py-0 font-semibold text-white">
      <span className="grid h-7 w-7 place-items-center rounded border border-[#52615b] text-[#65caa5]"><Clock3 size={16}/></span>
      <span>Codex Tempo</span>
    </Link>
    <nav className="grid gap-0.5 max-[760px]:flex max-[760px]:overflow-x-auto" aria-label="Main navigation">
      {links.map(([href, label, Icon]) => <Link className="flex min-h-9 items-center gap-2.5 rounded px-2.5 py-2 text-sm text-[#aebbb6] hover:bg-[#25302c] hover:text-white" href={href} key={href}><Icon size={16}/><span>{label}</span></Link>)}
    </nav>
    <div className="mt-auto border-t border-[#34413c] pt-4 text-xs text-[#7f8d88] max-[760px]:hidden">Tempo 0.1 · self-hosted</div>
  </aside>;
}
