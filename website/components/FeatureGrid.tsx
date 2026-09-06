"use client";

import { Terminal, Cloud, Play } from "lucide-react";
import { motion } from "framer-motion";

const features = [
  {
    icon: Terminal,
    title: "Secrets stay in memory",
    description: (
      <>
        <span className="font-mono text-zinc-300">inject</span> wraps your
        process. Secrets pass through OS environment inheritance and vanish when
        the terminal closes.
      </>
    ),
  },
  {
    icon: Cloud,
    title: "Your vault, your auth",
    description: (
      <>
        1Password and Bitwarden integration through their existing CLI sessions.
        No new accounts, tokens, or sync tools.
      </>
    ),
  },
  {
    icon: Play,
    title: "Your commands don’t change",
    description: (
      <>
        After setup, <span className="font-mono text-zinc-300">npm run dev</span>{" "}
        still works. <span className="font-mono text-zinc-300">inject</span> binds
        to your existing scripts.
      </>
    ),
  },
];

const containerVariants = {
  hidden: {},
  visible: {
    transition: {
      staggerChildren: 0.15,
    },
  },
};

const cardVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.5 } },
};

export default function FeatureGrid() {
  return (
    <section id="features" className="w-full border-t border-zinc-900 scroll-mt-24">
      <motion.div
        className="grid md:grid-cols-3 gap-6 max-w-6xl mx-auto px-4 py-24"
        variants={containerVariants}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-50px" }}
      >
        {features.map(({ icon: Icon, title, description }) => (
          <motion.div
            key={title}
            variants={cardVariants}
            className="bg-zinc-950 border border-zinc-800 rounded-xl p-6 hover:border-zinc-700 hover:-translate-y-1 hover:shadow-lg hover:shadow-zinc-900/50 transition-all duration-300 ease-out"
          >
            <div className="w-10 h-10 rounded-lg bg-zinc-900 border border-zinc-800 flex items-center justify-center mb-4">
              <Icon size={18} className="text-terminal-green" />
            </div>
            <h3 className="text-zinc-100 font-semibold mb-2 tracking-tight">
              {title}
            </h3>
            <p className="text-sm text-zinc-400 leading-relaxed">
              {description}
            </p>
          </motion.div>
        ))}
      </motion.div>
    </section>
  );
}
