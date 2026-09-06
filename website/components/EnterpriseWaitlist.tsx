"use client";

import { useState } from "react";
import { motion } from "framer-motion";

export default function EnterpriseWaitlist() {
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!email) return;
    setSubmitted(true);
  }

  return (
    <section id="enterprise" className="w-full bg-zinc-900/30 border-y border-zinc-800 py-24 scroll-mt-24">
      <div className="max-w-3xl mx-auto px-4 text-center">
        <p className="text-xs font-mono text-terminal-green uppercase tracking-widest mb-4">
          Enterprise
        </p>
        <h2 className="text-4xl md:text-5xl font-bold tracking-tighter text-zinc-100 mb-4 leading-tight">
          inject Enterprise
          <br />
          Control Plane
        </h2>
        <p className="text-lg text-zinc-400 max-w-xl mx-auto leading-relaxed">
          Bring comprehensive audit logs, RBAC, and policy enforcement to your
          developer&apos;s local environments.
        </p>

        {submitted ? (
          <div className="mt-8 inline-flex items-center gap-2 bg-zinc-900 border border-zinc-800 rounded-md px-5 py-3">
            <span className="text-terminal-green font-mono text-sm">{">"}</span>
            <span className="text-zinc-300 text-sm font-mono">
              You&apos;re on the list. We&apos;ll be in touch.
            </span>
          </div>
        ) : (
          <form
            onSubmit={handleSubmit}
            className="max-w-md mx-auto flex flex-col sm:flex-row mt-8 gap-2 sm:gap-0"
          >
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@company.com"
              className="bg-zinc-950 border border-zinc-800 sm:border-r-0 rounded-md sm:rounded-l-md sm:rounded-r-none px-4 py-2 text-sm text-zinc-100 placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-terminal-green/50 focus:border-terminal-green transition-all flex-grow min-w-0"
            />
            <motion.button
              type="submit"
              className="bg-zinc-100 text-zinc-950 text-sm font-mono px-4 py-2 rounded-md sm:rounded-l-none sm:rounded-r-md hover:bg-zinc-200 transition-colors duration-200 shrink-0 font-medium"
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              transition={{ type: "spring", stiffness: 300, damping: 20 }}
            >
              Join the Enterprise Waitlist
            </motion.button>
          </form>
        )}
      </div>
    </section>
  );
}
