"use client";

import { useState } from "react";
import HeroSection from "@/components/HeroSection";
import type { WorkflowMode } from "@/components/HeroSection";
import LogoStrip from "@/components/LogoStrip";
import FeatureGrid from "@/components/FeatureGrid";
import ArchitectureSection from "@/components/ArchitectureSection";
import GetStartedCTA from "@/components/GetStartedCTA";

const jsonLd = {
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  name: "inject",
  applicationCategory: "DeveloperApplication",
  operatingSystem: "macOS, Linux",
  offers: {
    "@type": "Offer",
    price: "0",
    priceCurrency: "USD",
  },
  description:
    "Zero-disk, zero-config secrets injection for local development. Fetch from 1Password, Bitwarden, or a local credential store.",
  url: "https://envpull.tech",
};

export default function Home() {
  const [workflowMode, setWorkflowMode] = useState<WorkflowMode>("local");

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <HeroSection workflowMode={workflowMode} setWorkflowMode={setWorkflowMode} />
      <LogoStrip />
      <FeatureGrid />
      <ArchitectureSection />
      <GetStartedCTA />
    </>
  );
}
