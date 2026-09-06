import { FileText } from "lucide-react";
import { siGithub } from "simple-icons";
import SimpleIconComponent from "@/components/ui/SimpleIcon";

export default function Footer() {
  return (
    <footer className="border-t border-zinc-900 py-12">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-between gap-4">
        <p className="text-xs text-zinc-500 font-mono">
          &copy; {new Date().getFullYear()} inject. All rights reserved.
        </p>

        <ul className="flex items-center gap-6">
          <li>
            <a
              href="https://github.com/dynamicHarsh/inject"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="GitHub"
              className="text-zinc-500 hover:text-zinc-300 transition-colors duration-200"
            >
              <SimpleIconComponent icon={siGithub} size={16} />
            </a>
          </li>
          <li>
            <a
              href="/getting-started"
              aria-label="Docs"
              className="text-zinc-500 hover:text-zinc-300 transition-colors duration-200"
            >
              <FileText size={16} />
            </a>
          </li>
        </ul>
      </div>
    </footer>
  );
}
