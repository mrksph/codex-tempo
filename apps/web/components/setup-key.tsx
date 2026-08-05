"use client";

import { Check, Copy, Eye, EyeOff } from "lucide-react";
import { useState } from "react";

export function SetupKey({ value }: { value: string }) {
  const [visible, setVisible] = useState(false);
  const [copied, setCopied] = useState(false);
  async function copy() {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  }
  return <div className="grid grid-cols-[minmax(0,1fr)_36px_36px] gap-1.5"><input aria-label="Setup key" className="min-w-0" readOnly type={visible ? "text" : "password"} value={value}/><button type="button" className="grid h-10 w-9 place-items-center cursor-pointer border border-[var(--line)] rounded bg-white text-[var(--muted)] hover:bg-[var(--surface-muted)] hover:text-[var(--ink)]" onClick={() => setVisible(!visible)} title={visible ? "Hide key" : "Show key"}>{visible ? <EyeOff size={15} /> : <Eye size={15} />}</button><button type="button" className="grid h-10 w-9 place-items-center cursor-pointer border border-[var(--line)] rounded bg-white text-[var(--muted)] hover:bg-[var(--surface-muted)] hover:text-[var(--ink)]" onClick={copy} title="Copy key">{copied ? <Check size={15} /> : <Copy size={15} />}</button></div>
}
