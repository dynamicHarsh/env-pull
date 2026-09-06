import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Getting Started",
};

function CodeBlock({ children }: { children: string }) {
  return (
    <pre className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 font-mono text-sm text-zinc-300 overflow-x-auto">
      {children}
    </pre>
  );
}

function SectionHeading({ children }: { children: React.ReactNode }) {
  return (
    <h2 className="text-3xl font-bold tracking-tighter text-zinc-100 mb-4 mt-16">
      {children}
    </h2>
  );
}

function Step({ n, children }: { n: number; children: React.ReactNode }) {
  return (
    <div className="flex gap-4 mt-6">
      <div className="w-7 h-7 rounded-full bg-zinc-800 border border-zinc-700 flex items-center justify-center shrink-0 mt-0.5">
        <span className="text-xs font-mono text-terminal-green">{n}</span>
      </div>
      <div className="flex-grow">{children}</div>
    </div>
  );
}

export default function GettingStartedPage() {
  return (
    <div className="max-w-3xl mx-auto px-4 py-24">
      <p className="text-xs font-mono text-terminal-green uppercase tracking-widest mb-4">
        Getting Started
      </p>
      <h1 className="text-4xl md:text-5xl font-bold tracking-tighter text-zinc-100 mb-6 leading-tight">
        Start using inject in 60 seconds.
      </h1>
      <p className="text-lg text-zinc-400 leading-relaxed mb-8">
        Install inject, run setup in your project, and your existing commands
        work the same way — with secrets injected securely from memory.
      </p>

      {/* Quickstart */}
      <SectionHeading>Quickstart</SectionHeading>
      <Step n={1}>
        <p className="text-zinc-300 mb-3">Install inject via Homebrew.</p>
        <CodeBlock>brew tap dynamicHarsh/tap && brew install inject</CodeBlock>
      </Step>
      <Step n={2}>
        <p className="text-zinc-300 mb-3">
          Run setup in your project directory. inject creates a project
          configuration and walks you through connecting a source.
        </p>
        <CodeBlock>inject setup</CodeBlock>
      </Step>
      <Step n={3}>
        <p className="text-zinc-300 mb-3">
          Run your usual commands. inject handles the rest.
        </p>
        <CodeBlock>npm run dev</CodeBlock>
      </Step>

      {/* Local .env example */}
      <SectionHeading>Local .env example</SectionHeading>
      <p className="text-zinc-400 leading-relaxed mb-2">
        You have an existing <code className="font-mono text-zinc-300">.env</code>{" "}
        file with project secrets. inject imports them into the credential store
        so you can delete the plaintext file.
      </p>
      <Step n={1}>
        <p className="text-zinc-300 mb-3">
          Run setup. inject detects the <code className="font-mono text-zinc-300">.env</code>{" "}
          file and offers to import its secrets.
        </p>
        <CodeBlock>{`inject setup\n# [inject] Found .env with 12 secrets.\n# [inject] Import into credential store? (Y/n)`}</CodeBlock>
      </Step>
      <Step n={2}>
        <p className="text-zinc-300 mb-3">
          Confirm the import. inject stores the secrets in the credential store,
          scoped to your project ID.
        </p>
        <CodeBlock>{`# [inject] Imported 12 secrets into credential store.\n# [inject] You can now safely delete .env.`}</CodeBlock>
      </Step>
      <Step n={3}>
        <p className="text-zinc-300 mb-3">
          Delete the plaintext <code className="font-mono text-zinc-300">.env</code>{" "}
          file and run your command as usual.
        </p>
        <CodeBlock>{`rm .env\nnpm run dev\n# > Ready on http://localhost:3000`}</CodeBlock>
      </Step>

      {/* 1Password team example */}
      <SectionHeading>1Password team example</SectionHeading>
      <p className="text-zinc-400 leading-relaxed mb-2">
        Your team stores secrets in a 1Password remote secret note. Each
        developer runs setup once to connect their local project to the shared
        vault.
      </p>
      <Step n={1}>
        <p className="text-zinc-300 mb-3">
          Make sure the 1Password CLI is installed and you have an active
          provider CLI session.
        </p>
        <CodeBlock>{`op signin`}</CodeBlock>
      </Step>
      <Step n={2}>
        <p className="text-zinc-300 mb-3">
          Run setup and select 1Password as the secret provider. inject reads the
          remote reference from the project configuration.
        </p>
        <CodeBlock>{`inject setup\n# [inject] Connected to 1Password vault "Engineering".`}</CodeBlock>
      </Step>
      <Step n={3}>
        <p className="text-zinc-300 mb-3">
          Run your command. inject fetches the secret set from 1Password via the
          existing CLI session and injects it into your process.
        </p>
        <CodeBlock>{`npm run dev\n# > Ready on http://localhost:3000`}</CodeBlock>
      </Step>

      {/* Bitwarden team example */}
      <SectionHeading>Bitwarden team example</SectionHeading>
      <p className="text-zinc-400 leading-relaxed mb-2">
        Same pattern as 1Password — your team stores secrets in a Bitwarden
        remote secret note, and inject reads from the Bitwarden CLI session.
      </p>
      <Step n={1}>
        <p className="text-zinc-300 mb-3">
          Make sure the Bitwarden CLI is installed and you have an active
          provider CLI session.
        </p>
        <CodeBlock>{`bw login && bw unlock`}</CodeBlock>
      </Step>
      <Step n={2}>
        <p className="text-zinc-300 mb-3">
          Run setup and select Bitwarden as the secret provider.
        </p>
        <CodeBlock>{`inject setup\n# [inject] Connected to Bitwarden vault "Engineering".`}</CodeBlock>
      </Step>
      <Step n={3}>
        <p className="text-zinc-300 mb-3">
          Run your command. inject fetches the secret set from Bitwarden and
          injects it into your process.
        </p>
        <CodeBlock>{`npm run dev\n# > Ready on http://localhost:3000`}</CodeBlock>
      </Step>
    </div>
  );
}
