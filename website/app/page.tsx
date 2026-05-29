import HeroSection from "@/components/HeroSection";
import FeatureGrid from "@/components/FeatureGrid";
import ArchitectureSection from "@/components/ArchitectureSection";
import EnterpriseWaitlist from "@/components/EnterpriseWaitlist";

const jsonLd = {
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  name: "env-pull",
  applicationCategory: "DeveloperApplication",
  operatingSystem: "macOS, Windows, Linux",
  offers: {
    "@type": "Offer",
    price: "0",
    priceCurrency: "USD",
  },
  description:
    "Zero-disk, zero-config secrets injection for local development. Fetch from AWS Secrets Manager, 1Password, or an encrypted local vault.",
  url: "https://envpull.tech",
};

export default function Home() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <HeroSection />
      <FeatureGrid />
      <ArchitectureSection />
      <EnterpriseWaitlist />
    </>
  );
}
