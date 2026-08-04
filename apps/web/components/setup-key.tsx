"use client";

import { Check, Copy, Eye, EyeOff } from "lucide-react";
import { useState } from "react";

export function SetupKey({ value }: { value: string }) {
  const [visible,setVisible]=useState(false);const [copied,setCopied]=useState(false);
  async function copy(){await navigator.clipboard.writeText(value);setCopied(true);window.setTimeout(()=>setCopied(false),1500)}
  return <div className="secret-row"><input aria-label="Setup key" readOnly type={visible?"text":"password"} value={value}/><button type="button" className="icon-button" onClick={()=>setVisible(!visible)} title={visible?"Hide key":"Show key"}>{visible?<EyeOff size={15}/>:<Eye size={15}/>}</button><button type="button" className="icon-button" onClick={copy} title="Copy key">{copied?<Check size={15}/>:<Copy size={15}/>}</button></div>
}
