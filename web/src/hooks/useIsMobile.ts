import { useEffect, useState } from "react";

const MOBILE_BREAKPOINT = 768; // md breakpoint in Tailwind CSS

/**
 * Hook to detect if the current viewport is mobile-sized.
 * Returns true for screens smaller than 768px (Tailwind's md breakpoint).
 *
 * This hook uses CSS media queries for accurate detection and
 * automatically updates on viewport resize.
 */
const useIsMobile = (): boolean => {
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window === "undefined") return false;
    return window.innerWidth < MOBILE_BREAKPOINT;
  });

  useEffect(() => {
    const mediaQuery = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);

    const handleChange = (e: MediaQueryListEvent) => {
      setIsMobile(e.matches);
    };

    // Set initial value based on media query
    setIsMobile(mediaQuery.matches);

    mediaQuery.addEventListener("change", handleChange);

    return () => {
      mediaQuery.removeEventListener("change", handleChange);
    };
  }, []);

  return isMobile;
};

export default useIsMobile;
