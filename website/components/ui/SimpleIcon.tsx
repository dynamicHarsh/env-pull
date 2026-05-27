import type { SimpleIcon } from "simple-icons";

interface SimpleIconProps {
  icon: SimpleIcon;
  size?: number;
  className?: string;
  /** Use brand color. Defaults to false (inherits currentColor). */
  branded?: boolean;
}

export default function SimpleIconComponent({
  icon,
  size = 16,
  className = "",
  branded = false,
}: SimpleIconProps) {
  return (
    <svg
      role="img"
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill={branded ? `#${icon.hex}` : "currentColor"}
      aria-label={icon.title}
      className={className}
      xmlns="http://www.w3.org/2000/svg"
    >
      <path d={icon.path} />
    </svg>
  );
}
