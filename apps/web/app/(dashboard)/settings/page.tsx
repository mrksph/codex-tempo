import { PageHeader } from "@/components/page-header";
import { SetupKey } from "@/components/setup-key";

export const dynamic = "force-dynamic";

export default function SettingsPage() {
  const setupKey = process.env.AGENT_SETUP_KEY || process.env.ADMIN_TOKEN || "Not configured";
  const serverURL = process.env.PUBLIC_API_URL || "http://localhost:8080";

  return (
    <>
      <PageHeader title="Settings" subtitle="Agents and privacy" />
      <div className="grid gap-6">
        <section className="border border-[var(--line)] bg-[var(--surface)]">
          <div className="flex min-h-[52px] items-center border-b border-[var(--line)] px-4 py-3.5">
            <h2 className="text-sm font-semibold tracking-normal">Agent registration</h2>
          </div>
          <div className="grid gap-4 p-4 text-[13px]">
            <label className="grid gap-1.5">
              <span>Server URL</span>
              <input className="h-10 border border-[var(--line)] px-2.5" value={serverURL} readOnly />
            </label>
            <label className="grid gap-1.5">
              Setup key
              <SetupKey value={setupKey} />
            </label>
          </div>
        </section>

        <section className="border border-[var(--line)] bg-[var(--surface)]">
          <div className="flex min-h-[52px] items-center border-b border-[var(--line)] px-4 py-3.5">
            <h2 className="text-sm font-semibold tracking-normal">Privacy</h2>
          </div>
          <div className="grid gap-3.5 p-4 text-[13px]">
            <label className="flex items-center justify-between gap-5">Store project paths <input type="checkbox" disabled /></label>
            <label className="flex items-center justify-between gap-5">Store tool names <input type="checkbox" defaultChecked disabled /></label>
            <label className="flex items-center justify-between gap-5">Store content <input type="checkbox" disabled /></label>
          </div>
        </section>
      </div>
    </>
  );
}
