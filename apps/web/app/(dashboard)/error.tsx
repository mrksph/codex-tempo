"use client";
import { RefreshCw } from "lucide-react";
export default function ErrorPage({reset}:{error:Error;reset:()=>void}){return <div className="border border-[#e1bbbb] bg-[#fff7f7] p-4 text-sm text-[var(--danger)]"><RefreshCw size={14}/>Could not load server data. <button className="ml-3 inline-flex min-h-9 items-center justify-center rounded bg-[var(--accent)] px-3 text-xs font-bold text-white" onClick={reset}><span>Retry</span></button></div>}
