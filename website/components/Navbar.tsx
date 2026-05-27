"use client";

import { useState } from "react";
import Link from "next/link";
import { Menu, X } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";

const navLinks = [
  { label: "Features", href: "#features" },
  { label: "Security", href: "#security" },
  { label: "Enterprise", href: "#enterprise" },
];

export default function Navbar() {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [active, setActive] = useState<string>("");

  return (
    <header className="sticky top-0 z-50 bg-zinc-950/70 backdrop-blur-md border-b border-zinc-800/50">
      <nav className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
        {/* Brand */}
        <Link href="/" className="flex items-center gap-0 shrink-0">
          <span className="font-mono font-bold text-zinc-100 tracking-tight text-sm">
            env-pull
          </span>
          <span className="w-2 h-4 bg-terminal-green animate-pulse inline-block ml-1 align-middle" />
        </Link>

        {/* Desktop links */}
        <ul className="hidden md:flex items-center gap-1">
          {navLinks.map((link) => (
            <li key={link.label}>
              <a
                href={link.href}
                onClick={() => setActive(link.label)}
                className="relative px-3 py-1.5 text-sm text-zinc-400 hover:text-zinc-100 transition-colors duration-200 flex items-center z-10"
              >
                {active === link.label && (
                  <motion.div
                    layoutId="nav-slider"
                    className="absolute inset-0 bg-zinc-800/50 rounded-md -z-10"
                    transition={{ type: "spring", stiffness: 300, damping: 30 }}
                  />
                )}
                {active === link.label && (
                  <span className="text-terminal-green font-mono mr-2">{">"}</span>
                )}
                {link.label}
              </a>
            </li>
          ))}
        </ul>

        {/* Desktop CTA */}
        <a
          href="#enterprise"
          className="hidden md:inline-flex items-center bg-[#f4f4f5] text-zinc-950 px-4 py-2 rounded-md font-mono text-sm font-semibold hover:bg-[#e4e4e7] transition-colors shrink-0"
        >
          Get Started
        </a>

        {/* Mobile hamburger */}
        <button
          onClick={() => setMobileOpen((v) => !v)}
          className="md:hidden text-zinc-400 hover:text-zinc-100 transition-colors"
          aria-label="Toggle menu"
        >
          {mobileOpen ? <X size={20} /> : <Menu size={20} />}
        </button>
      </nav>

      {/* Mobile menu */}
      <AnimatePresence>
        {mobileOpen && (
          <motion.div
            key="mobile-menu"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2, ease: "easeInOut" }}
            className="md:hidden overflow-hidden border-t border-zinc-800/50 bg-zinc-950/95"
          >
            <ul className="flex flex-col px-4 py-4 gap-1">
              {navLinks.map((link) => (
                <li key={link.label}>
                  <a
                    href={link.href}
                    onClick={() => {
                      setActive(link.label);
                      setMobileOpen(false);
                    }}
                    className="relative flex items-center text-sm text-zinc-400 hover:text-zinc-100 transition-colors duration-200 px-3 py-2 rounded-md hover:bg-zinc-800/40"
                  >
                    {active === link.label && (
                      <span className="text-terminal-green font-mono mr-2">{">"}</span>
                    )}
                    {link.label}
                  </a>
                </li>
              ))}
              <li className="pt-3 border-t border-zinc-800/50 mt-1">
                <a
                  href="#enterprise"
                  onClick={() => setMobileOpen(false)}
                  className="inline-flex items-center bg-[#f4f4f5] text-zinc-950 px-4 py-2 rounded-md font-mono text-sm font-semibold hover:bg-[#e4e4e7] transition-colors"
                >
                  Get Started
                </a>
              </li>
            </ul>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  );
}
