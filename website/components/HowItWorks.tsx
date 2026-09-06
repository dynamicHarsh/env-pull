"use client";

import { FileDown, Play, Trash2, KeyRound } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { WorkflowMode } from "@/components/HeroSection";

const sharedSteps = [
  {
    number: 2,
    icon: Play,
    title: "Run your usual command",
    description: (
      <>
        <span className="font-mono text-zinc-300">inject</span> wraps your
        process. Secrets load into memory and your command runs as normal.
      </>
    ),
  },
  {
    number: 3,
    icon: Trash2,
    title: "Delete your .env",
    description: (
      <>
        No plaintext files on disk. Secrets live only in the process tree and
        vanish when it exits.
      </>
    ),
  },
];

const step1: Record<
  WorkflowMode,
  { icon: typeof FileDown; title: string; description: React.ReactNode }
> = {
  local: {
    icon: FileDown,
    title: "Import your .env",
    description: (
      <>
        Point <span className="font-mono text-zinc-300">inject</span> at your
        existing <span className="font-mono text-zinc-300">.env</span> file to
        encrypt and store it locally.
      </>
    ),
  },
  onepassword: {
    icon: KeyRound,
    title: "Connect your vault",
    description: (
      <>
        Authenticate with 1Password via its CLI.{" "}
        <span className="font-mono text-zinc-300">inject</span> reads secrets
        straight from your vault.
      </>
    ),
  },
};

const containerVariants = {
  hidden: {},
  visible: {
    transition: {
      staggerChildren: 0.15,
    },
  },
};

const stepVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.5 } },
};

const contentVariants = {
  initial: { opacity: 0, y: 4 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.25 } },
  exit: { opacity: 0, y: -4, transition: { duration: 0.15 } },
};

export default function HowItWorks({
  workflowMode,
}: {
  workflowMode: WorkflowMode;
}) {
  const first = step1[workflowMode];
  const steps = [
    { number: 1, icon: first.icon, title: first.title, description: first.description },
    ...sharedSteps,
  ];

  return (
    <section
      id="how-it-works"
      className="w-full border-t border-zinc-900 scroll-mt-24"
    >
      <div className="max-w-6xl mx-auto px-4 py-24 text-center">
        <p className="text-xs font-mono text-terminal-green uppercase tracking-widest mb-4">
          How it works
        </p>
        <h2 className="text-4xl md:text-5xl font-bold tracking-tighter text-zinc-100 mb-6 leading-tight">
          Three steps. Zero secrets on disk.
        </h2>
        <p className="text-lg text-zinc-400 leading-relaxed max-w-2xl mx-auto mb-16">
          Whether you start from a local file or a cloud vault, the workflow is
          the same.
        </p>

        <motion.div
          className="grid md:grid-cols-3 gap-6"
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-50px" }}
        >
          {steps.map((step) => (
            <motion.div
              key={step.number}
              variants={stepVariants}
              className="relative bg-zinc-950 border border-zinc-800 rounded-xl p-6 text-left"
            >
              <div className="flex items-center gap-3 mb-4">
                <span className="flex-shrink-0 w-8 h-8 rounded-full bg-terminal-green/10 border border-terminal-green/20 flex items-center justify-center text-sm font-mono font-bold text-terminal-green">
                  {step.number}
                </span>
                <AnimatePresence mode="wait">
                  <motion.div
                    key={step.title}
                    variants={contentVariants}
                    initial="initial"
                    animate="animate"
                    exit="exit"
                    className="flex items-center gap-2"
                  >
                    <step.icon size={16} className="text-zinc-400" />
                    <h3 className="text-zinc-100 font-semibold tracking-tight">
                      {step.title}
                    </h3>
                  </motion.div>
                </AnimatePresence>
              </div>
              <AnimatePresence mode="wait">
                <motion.p
                  key={step.title}
                  variants={contentVariants}
                  initial="initial"
                  animate="animate"
                  exit="exit"
                  className="text-sm text-zinc-400 leading-relaxed"
                >
                  {step.description}
                </motion.p>
              </AnimatePresence>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
