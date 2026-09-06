import SimpleIconComponent from "@/components/ui/SimpleIcon";

let si1passwordIcon: import("simple-icons").SimpleIcon | null = null;
let siBitwardenIcon: import("simple-icons").SimpleIcon | null = null;

try {
  const si = require("simple-icons");
  si1passwordIcon = si.si1password ?? null;
  siBitwardenIcon = si.siBitwarden ?? null;
} catch {
  // simple-icons unavailable or slugs missing — fall back to inline SVGs below
}

function OnePasswordFallback({ size }: { size: number }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="currentColor" aria-label="1Password" xmlns="http://www.w3.org/2000/svg">
      <path d="M12 0C5.373 0 0 5.373 0 12s5.373 12 12 12 12-5.373 12-12S18.627 0 12 0zm-.6 4.8h1.2c.31 0 .56.25.56.56v8.08c0 .31-.25.56-.56.56h-1.2a.56.56 0 0 1-.56-.56V5.36c0-.31.25-.56.56-.56z" />
    </svg>
  );
}

function BitwardenFallback({ size }: { size: number }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="currentColor" aria-label="Bitwarden" xmlns="http://www.w3.org/2000/svg">
      <path d="M3.5 3.5v12.21c0 .36.18.7.48.9L12 21.5l8.02-4.89c.3-.2.48-.54.48-.9V3.5L12 .5 3.5 3.5zm15 11.86L12 19.5l-6.5-4.14V5.5L12 2.5l6.5 3v9.86z" />
    </svg>
  );
}

export default function LogoStrip() {
  return (
    <div className="w-full border-t border-zinc-900 py-8">
      <div className="max-w-6xl mx-auto px-4 flex items-center justify-center gap-6">
        <span className="text-xs text-zinc-500 font-mono">Works with</span>
        <div className="flex items-center gap-5">
          <div className="flex items-center gap-2 text-zinc-500">
            {si1passwordIcon ? (
              <SimpleIconComponent icon={si1passwordIcon} size={18} />
            ) : (
              <OnePasswordFallback size={18} />
            )}
            <span className="text-xs">1Password</span>
          </div>
          <div className="flex items-center gap-2 text-zinc-500">
            {siBitwardenIcon ? (
              <SimpleIconComponent icon={siBitwardenIcon} size={18} />
            ) : (
              <BitwardenFallback size={18} />
            )}
            <span className="text-xs">Bitwarden</span>
          </div>
        </div>
      </div>
    </div>
  );
}
