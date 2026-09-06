"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import type { WorkflowMode } from "@/components/HeroSection";

const SEQUENCES: Record<WorkflowMode, { command: string; response: string; second: string; ready: string }> = {
  local: {
    command: "$ inject setup",
    response: '[inject] Imported 12 secrets from .env into credential store.',
    second: "$ npm run dev",
    ready: "> Ready on http://localhost:3000",
  },
  onepassword: {
    command: "$ inject setup",
    response: '[inject] Connected to 1Password vault "Engineering".',
    second: "$ npm run dev",
    ready: "> Ready on http://localhost:3000",
  },
};

const CHAR_DELAY_MS = 42;

interface TerminalAnimationProps {
  workflowMode: WorkflowMode;
}

export default function TerminalAnimation({ workflowMode }: TerminalAnimationProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [hasStarted, setHasStarted] = useState(false);
  const [typed, setTyped] = useState("");
  const [phase, setPhase] = useState<"idle" | "response" | "second" | "ready">("idle");
  const [secondTyped, setSecondTyped] = useState("");
  const animationRef = useRef<number>(0);

  const seq = SEQUENCES[workflowMode];

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

  const resetAnimation = useCallback(() => {
    animationRef.current++;
    setTyped("");
    setSecondTyped("");
    setPhase("idle");
  }, []);

  useEffect(() => {
    resetAnimation();
  }, [workflowMode, resetAnimation]);

  useEffect(() => {
    if (!hasStarted) return;

    const runId = animationRef.current;
    let charIndex = 0;
    const command = seq.command;

    const interval = setInterval(() => {
      if (runId !== animationRef.current) {
        clearInterval(interval);
        return;
      }
      charIndex++;
      setTyped(command.slice(0, charIndex));
      if (charIndex === command.length) {
        clearInterval(interval);
        setTimeout(() => {
          if (runId === animationRef.current) setPhase("response");
        }, 380);
      }
    }, CHAR_DELAY_MS);

    return () => clearInterval(interval);
  }, [hasStarted, workflowMode, seq.command]);

  useEffect(() => {
    if (phase !== "response") return;

    const runId = animationRef.current;
    const t = setTimeout(() => {
      if (runId !== animationRef.current) return;
      setPhase("second");
    }, 700);
    return () => clearTimeout(t);
  }, [phase]);

  useEffect(() => {
    if (phase !== "second") return;

    const runId = animationRef.current;
    let charIndex = 0;
    const command = seq.second;

    const interval = setInterval(() => {
      if (runId !== animationRef.current) {
        clearInterval(interval);
        return;
      }
      charIndex++;
      setSecondTyped(command.slice(0, charIndex));
      if (charIndex === command.length) {
        clearInterval(interval);
        setTimeout(() => {
          if (runId === animationRef.current) setPhase("ready");
        }, 380);
      }
    }, CHAR_DELAY_MS);

    return () => clearInterval(interval);
  }, [phase, seq.second]);

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
          bash — inject
        </span>
      </div>

      {/* Terminal body */}
      <div className="p-5 font-mono text-sm leading-relaxed min-h-[170px]">
        {/* First command */}
        <div>
          <span className="text-zinc-300">{typed}</span>
          {phase === "idle" && typed.length > 0 && typed.length < seq.command.length && (
            <span className="animate-pulse text-terminal-green ml-px">▌</span>
          )}
        </div>

        {/* Setup response */}
        <AnimatePresence>
          {(phase === "response" || phase === "second" || phase === "ready") && (
            <motion.div
              key={`response-${workflowMode}`}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.35 }}
              className="mt-2 text-terminal-green"
            >
              {seq.response}
            </motion.div>
          )}
        </AnimatePresence>

        {/* Second command */}
        <AnimatePresence>
          {(phase === "second" || phase === "ready") && (
            <motion.div
              key={`second-${workflowMode}`}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.35 }}
              className="mt-2"
            >
              <span className="text-zinc-300">{secondTyped}</span>
              {phase === "second" && secondTyped.length < seq.second.length && (
                <span className="animate-pulse text-terminal-green ml-px">▌</span>
              )}
            </motion.div>
          )}
        </AnimatePresence>

        {/* Ready output */}
        <AnimatePresence>
          {phase === "ready" && (
            <motion.div
              key={`ready-${workflowMode}`}
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
