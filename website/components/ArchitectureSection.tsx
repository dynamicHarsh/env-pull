"use client";

import { Lock, Cpu, AppWindow, X } from "lucide-react";
import { motion } from "framer-motion";

function FlowNode({
  icon: Icon,
  label,
  sublabel,
  accent = false,
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  sublabel?: string;
  accent?: boolean;
}) {
  if (accent) {
    return (
      <motion.div
        className="border border-terminal-green/40 rounded-lg p-4 bg-zinc-900 flex items-center gap-3"
        animate={{
          boxShadow: [
            "0px 0px 0px rgba(57,255,20,0)",
            "0px 0px 15px rgba(57,255,20,0.15)",
            "0px 0px 0px rgba(57,255,20,0)",
          ],
        }}
        transition={{ repeat: Infinity, duration: 3 }}
      >
        <div className="w-8 h-8 rounded-md flex items-center justify-center shrink-0 bg-terminal-green/10">
          <Icon size={16} className="text-terminal-green" />
        </div>
        <div>
          <p className="text-sm text-zinc-100 font-mono font-medium leading-tight">
            {label}
          </p>
          {sublabel && (
            <p className="text-xs text-zinc-500 leading-tight mt-0.5">
              {sublabel}
            </p>
          )}
        </div>
      </motion.div>
    );
  }

  return (
    <div className="border border-zinc-800 rounded-lg p-4 bg-zinc-900 flex items-center gap-3">
      <div className="w-8 h-8 rounded-md flex items-center justify-center shrink-0 bg-zinc-800">
        <Icon size={16} className="text-zinc-400" />
      </div>
      <div>
        <p className="text-sm text-zinc-100 font-mono font-medium leading-tight">
          {label}
        </p>
        {sublabel && (
          <p className="text-xs text-zinc-500 leading-tight mt-0.5">
            {sublabel}
          </p>
        )}
      </div>
    </div>
  );
}

function Arrow() {
  return (
    <div className="flex items-center justify-center py-1">
      <div className="w-px h-5 border-l border-dashed border-zinc-700" />
    </div>
  );
}

function BypassedNode() {
  return (
    <div className="flex items-center gap-3 relative pl-2">
      <div className="w-px h-full absolute left-[1.85rem] top-0 border-l border-dashed border-zinc-800" />
      <div className="border border-red-900/50 rounded-lg p-3 bg-red-950/20 flex items-center gap-3 opacity-50 w-full">
        <div className="w-8 h-8 rounded-md flex items-center justify-center shrink-0 bg-red-950/40">
          <X size={14} className="text-red-500" />
        </div>
        <div>
          <p className="text-sm text-red-400 font-mono font-medium leading-tight line-through">
            Disk / .env
          </p>
          <p className="text-xs text-red-600 leading-tight mt-0.5">
            never touched
          </p>
        </div>
      </div>
    </div>
  );
}

const slideInLeft = {
  hidden: { opacity: 0, x: -30 },
  visible: { opacity: 1, x: 0, transition: { type: "spring" as const, stiffness: 80, damping: 20 } },
};

const slideInRight = {
  hidden: { opacity: 0, x: 30 },
  visible: { opacity: 1, x: 0, transition: { type: "spring" as const, stiffness: 80, damping: 20 } },
};

export default function ArchitectureSection() {
  return (
    <section id="security" className="w-full border-t border-zinc-900 overflow-hidden scroll-mt-24">
      <div className="grid md:grid-cols-2 items-center gap-12 py-24 max-w-6xl mx-auto px-4">
        {/* Copy */}
        <motion.div
          variants={slideInLeft}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-50px" }}
        >
          <p className="text-xs font-mono text-terminal-green uppercase tracking-widest mb-4">
            Security Model
          </p>
          <h2 className="text-4xl md:text-5xl font-bold tracking-tighter text-zinc-100 mb-6 leading-tight">
            Eliminate the #1 cause of leaked credentials.
          </h2>
          <p className="text-lg text-zinc-400 leading-relaxed mb-8">
            No plaintext{" "}
            <span className="font-mono text-zinc-300">.env</span> files on
            developer laptops, ever.
          </p>
          <ul className="space-y-3">
            {[
              "Secrets live exclusively in process memory",
              "Zero footprint on the filesystem",
              "Automatic cleanup on process exit",
              "Auditable vault access via existing toolchain",
            ].map((point) => (
              <li key={point} className="flex items-start gap-2 text-sm text-zinc-400">
                <span className="text-terminal-green font-mono mt-0.5 shrink-0">{">"}</span>
                {point}
              </li>
            ))}
          </ul>
        </motion.div>

        {/* Architecture diagram */}
        <motion.div
          className="flex flex-col max-w-xs mx-auto w-full gap-0"
          variants={slideInRight}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-50px" }}
        >
          <FlowNode
            icon={Lock}
            label="Upstream Vault"
            sublabel="AWS Secrets Manager · 1Password"
          />
          <Arrow />
          <FlowNode
            icon={Cpu}
            label="inject (Memory)"
            sublabel="secrets injected via process env"
            accent
          />
          <Arrow />
          <FlowNode
            icon={AppWindow}
            label="Local Process"
            sublabel="npm run dev · any command"
          />
          <div className="mt-4 pt-4 border-t border-zinc-800/50">
            <BypassedNode />
          </div>
        </motion.div>
      </div>
    </section>
  );
}
