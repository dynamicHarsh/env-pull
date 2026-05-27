"use client";

import { useEffect, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";

const COMMAND = "$ env-pull run --aws-secret npm run dev";
const CHAR_DELAY_MS = 42;

export default function TerminalAnimation() {
  const containerRef = useRef<HTMLDivElement>(null);
  const [hasStarted, setHasStarted] = useState(false);
  const [typed, setTyped] = useState("");
  const [phase, setPhase] = useState<"idle" | "injected" | "ready">("idle");

  // Start animation when terminal enters the viewport
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setHasStarted(true);
          observer.disconnect();
        }
      },
      { threshold: 0.4 }
    );

    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  // Typing sequence — only runs after viewport entry
  useEffect(() => {
    if (!hasStarted) return;

    let charIndex = 0;
    const interval = setInterval(() => {
      charIndex++;
      setTyped(COMMAND.slice(0, charIndex));
      if (charIndex === COMMAND.length) {
        clearInterval(interval);
        setTimeout(() => setPhase("injected"), 380);
      }
    }, CHAR_DELAY_MS);

    return () => clearInterval(interval);
  }, [hasStarted]);

  useEffect(() => {
    if (phase === "injected") {
      const t = setTimeout(() => setPhase("ready"), 700);
      return () => clearTimeout(t);
    }
  }, [phase]);

  return (
    <div
      ref={containerRef}
      className="bg-zinc-950 border border-zinc-800 rounded-lg shadow-2xl overflow-hidden max-w-2xl w-full mx-auto"
    >
      {/* macOS-style top bar */}
      <div className="flex items-center px-4 h-9 bg-zinc-900/80 border-b border-zinc-800 gap-2">
        <span className="w-3 h-3 rounded-full bg-zinc-700/80" />
        <span className="w-3 h-3 rounded-full bg-zinc-700/80" />
        <span className="w-3 h-3 rounded-full bg-zinc-700/80" />
        <span className="flex-1 text-center text-xs font-mono text-zinc-500 select-none">
          bash — env-pull
        </span>
      </div>

      {/* Terminal body */}
      <div className="p-5 font-mono text-sm leading-relaxed min-h-[130px]">
        {/* Command line */}
        <div>
          <span className="text-zinc-300">{typed}</span>
          {phase === "idle" && (
            <span className="animate-pulse text-terminal-green ml-px">▌</span>
          )}
        </div>

        {/* Injection success */}
        <AnimatePresence>
          {(phase === "injected" || phase === "ready") && (
            <motion.div
              key="injected"
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.35 }}
              className="mt-2 text-terminal-green"
            >
              [env-pull] Injected 14 secrets into memory.
            </motion.div>
          )}
        </AnimatePresence>

        {/* Server ready */}
        <AnimatePresence>
          {phase === "ready" && (
            <motion.div
              key="ready"
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.35 }}
              className="mt-1 text-zinc-400"
            >
              <span className="text-terminal-green font-mono mr-2">{">"}</span>
              Ready on{" "}
              <span className="text-zinc-300">http://localhost:3000</span>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
