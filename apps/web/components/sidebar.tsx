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
  return <aside className="sidebar">
    <Link href="/dashboard" className="brand"><span className="brand-mark"><Clock3 size={16}/></span><span>Codex Tempo</span></Link>
    <nav className="nav" aria-label="Main navigation">
      {links.map(([href, label, Icon]) => <Link className="nav-link" href={href} key={href}><Icon size={16}/><span>{label}</span></Link>)}
    </nav>
    <div className="sidebar-foot">Tempo 0.1 · self-hosted</div>
  </aside>;
}
