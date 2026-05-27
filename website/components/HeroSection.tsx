"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { Check, Copy } from "lucide-react";
import TerminalAnimation from "@/components/TerminalAnimation";

const commands = {
  mac: "brew tap dynamicHarsh/tap && brew install env-pull",
  windows:
    "scoop bucket add env-pull https://github.com/dynamicHarsh/scoop-bucket && scoop install env-pull",
};

export default function HeroSection() {
  const [os, setOs] = useState<"mac" | "windows">("mac");
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    const text = commands[os];
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text).catch(() => fallbackCopy(text));
    } else {
      fallbackCopy(text);
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  function fallbackCopy(text: string) {
    const el = document.createElement("textarea");
    el.value = text;
    el.style.cssText = "position:fixed;opacity:0";
    document.body.appendChild(el);
    el.focus();
    el.select();
    document.execCommand("copy");
    document.body.removeChild(el);
  }

  return (
    <section className="w-full flex flex-col items-center justify-center pt-32 pb-20 px-4 text-center">
      {/* Headline */}
      <h1 className="text-5xl md:text-7xl font-bold tracking-tighter bg-gradient-to-br from-zinc-100 to-zinc-500 bg-clip-text text-transparent mb-6 leading-[1.1]">
        The Universal
        <br />
        Secrets Adapter
      </h1>

      {/* Sub-headline */}
      <p className="text-lg md:text-xl text-zinc-400 max-w-2xl mx-auto mb-10 leading-relaxed">
        Zero-disk, zero-config secrets injection for local development.{" "}
        <br className="hidden sm:block" />
        Stop shuffling <span className="font-mono text-zinc-300">.env</span>{" "}
        files.
      </p>

      {/* OS toggle */}
      <div className="flex items-center justify-center space-x-1 bg-zinc-900/50 p-1 rounded-full mb-3 w-max mx-auto border border-zinc-800">
        {(["mac", "windows"] as const).map((platform) => (
          <button
            key={platform}
            onClick={() => setOs(platform)}
            className="relative px-3 py-1 text-xs font-medium rounded-full transition-colors z-10"
          >
            {os === platform && (
              <motion.div
                layoutId="os-slider"
                className="absolute inset-0 bg-zinc-700/50 rounded-full -z-10"
                transition={{ type: "spring", stiffness: 300, damping: 30 }}
              />
            )}
            <span className={os === platform ? "text-zinc-100" : "text-zinc-500 hover:text-zinc-300"}>
              {platform === "mac" ? "macOS" : "Windows"}
            </span>
          </button>
        ))}
      </div>

      {/* Install pill */}
      <div className="flex items-center justify-between bg-[#18181b] border border-zinc-800 rounded-full px-5 py-3 w-full max-w-2xl mx-auto overflow-hidden mb-16">
        <span className="font-mono text-sm text-zinc-300 overflow-x-auto whitespace-nowrap scrollbar-hide flex-grow mr-4">
          {commands[os]}
        </span>
        <motion.button
          onClick={handleCopy}
          aria-label="Copy install command"
          className="shrink-0 text-zinc-500 hover:text-zinc-300 transition-colors"
          whileHover={{ scale: 1.1 }}
          whileTap={{ scale: 0.9 }}
          transition={{ type: "spring", stiffness: 400, damping: 17 }}
        >
          {copied ? (
            <Check className="w-4 h-4 text-terminal-green" />
          ) : (
            <Copy className="w-4 h-4" />
          )}
        </motion.button>
      </div>

      {/* Terminal demo */}
      <TerminalAnimation />
    </section>
  );
}
