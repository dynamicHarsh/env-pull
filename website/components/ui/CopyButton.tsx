"use client";

import { useState } from "react";
import { Copy, Check } from "lucide-react";

interface CopyButtonProps {
  text: string;
}

export default function CopyButton({ text }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <button
      onClick={handleCopy}
      aria-label="Copy install command"
      className="text-zinc-500 hover:text-zinc-300 cursor-pointer transition-colors duration-200 ml-3 shrink-0"
    >
      {copied ? (
        <Check size={14} className="text-terminal-green" />
      ) : (
        <Copy size={14} />
      )}
    </button>
  );
}
