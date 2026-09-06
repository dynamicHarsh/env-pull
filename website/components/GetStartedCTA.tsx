"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { Check, Copy, ArrowRight } from "lucide-react";
import Link from "next/link";
import SectionDivider from "@/components/SectionDivider";

const INSTALL_CMD = "brew tap dynamicHarsh/tap && brew install inject";

export default function GetStartedCTA() {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(INSTALL_CMD).catch(() => fallbackCopy());
    } else {
      fallbackCopy();
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  function fallbackCopy() {
    const el = document.createElement("textarea");
    el.value = INSTALL_CMD;
    el.style.cssText = "position:fixed;opacity:0";
    document.body.appendChild(el);
    el.focus();
    el.select();
    document.execCommand("copy");
    document.body.removeChild(el);
  }

  return (
    <section id="get-started" className="w-full bg-zinc-900/30 scroll-mt-24">
      <SectionDivider />
      <div className="max-w-3xl mx-auto px-4 text-center py-24">
        <p className="text-xs font-mono text-terminal-green uppercase tracking-widest mb-4">
          Get Started
        </p>
        <h2 className="text-4xl md:text-5xl font-bold tracking-tighter text-zinc-100 mb-8 leading-tight">
          Install inject
        </h2>

        {/* Install pill */}
        <div className="flex items-center justify-between bg-[#18181b] border border-zinc-800 rounded-full px-5 py-3 w-full max-w-2xl mx-auto overflow-hidden mb-8">
          <span className="font-mono text-sm text-zinc-300 overflow-x-auto whitespace-nowrap scrollbar-hide flex-grow mr-4">
            {INSTALL_CMD}
          </span>
          <motion.button
            onClick={handleCopy}
            aria-label="Copy install command"
            className="shrink-0 text-zinc-500 hover:text-zinc-300 transition-colors cursor-pointer"
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

        <Link
          href="/getting-started"
          className="inline-flex items-center gap-2 text-terminal-green hover:text-emerald-400 font-mono text-sm transition-colors"
        >
          Read the Getting Started guide
          <ArrowRight size={14} />
        </Link>
      </div>
      <SectionDivider />
    </section>
  );
}
