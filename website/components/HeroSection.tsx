"use client";

import { motion } from "framer-motion";
import TerminalAnimation from "@/components/TerminalAnimation";
import CopyButton from "@/components/ui/CopyButton";

const INSTALL_COMMAND = "brew install env-pull";

export default function HeroSection() {
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

      {/* Install CTA */}
      <motion.div
        className="bg-zinc-900 border border-zinc-800 rounded-full flex items-center px-5 py-2.5 mb-16 max-w-sm w-full"
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.98 }}
        transition={{ type: "spring", stiffness: 300, damping: 20 }}
      >
        <span className="flex-1 font-mono text-sm text-zinc-300 text-left select-all">
          {INSTALL_COMMAND}
        </span>
        <CopyButton text={INSTALL_COMMAND} />
      </motion.div>

      {/* Terminal demo */}
      <TerminalAnimation />
    </section>
  );
}
