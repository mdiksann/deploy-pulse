import React, { useEffect, useRef, useState } from "react";

/**
 * Hook to track whether an element is visible in the viewport using IntersectionObserver.
 * @param {Object} options - IntersectionObserver options
 * @returns {[React.RefObject, boolean]} [ref, isVisible]
 */
export function useScrollReveal(options = { threshold: 0.15, rootMargin: "0px 0px -50px 0px" }) {
  const ref = useRef(null);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    if (typeof window === "undefined" || !("IntersectionObserver" in window)) {
      setIsVisible(true);
      return;
    }

    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) {
        setIsVisible(true);
        observer.unobserve(entry.target);
      }
    }, options);

    observer.observe(el);
    return () => {
      if (el) observer.unobserve(el);
    };
  }, [options.threshold, options.rootMargin]);

  return [ref, isVisible];
}

/**
 * Component that animates number counting when scrolled into view.
 */
export function AnimatedCounter({ value, suffix = "", prefix = "", decimals = 0, duration = 1600 }) {
  const [ref, isVisible] = useScrollReveal();
  const [displayValue, setDisplayValue] = useState(0);

  useEffect(() => {
    if (!isVisible) return;

    const numericTarget = typeof value === "number" ? value : parseFloat(value) || 0;
    const startTime = performance.now();

    let frameId;
    const updateCount = (currentTime) => {
      const elapsed = currentTime - startTime;
      const progress = Math.min(elapsed / duration, 1);
      // Ease out cubic
      const easeOut = 1 - Math.pow(1 - progress, 3);
      const current = numericTarget * easeOut;

      setDisplayValue(current);

      if (progress < 1) {
        frameId = requestAnimationFrame(updateCount);
      } else {
        setDisplayValue(numericTarget);
      }
    };

    frameId = requestAnimationFrame(updateCount);
    return () => {
      if (frameId) cancelAnimationFrame(frameId);
    };
  }, [isVisible, value, duration]);

  const formatted = decimals > 0 ? displayValue.toFixed(decimals) : Math.round(displayValue);

  return (
    <span ref={ref} className="animated-counter">
      {prefix}{isVisible ? formatted : (decimals > 0 ? (0).toFixed(decimals) : 0)}{suffix}
    </span>
  );
}

/**
 * Container component that animates its children when scrolled into view.
 */
export function ScrollReveal({
  children,
  className = "",
  delay = 0,
  direction = "up",
  distance = "24px",
  style = {},
  ...props
}) {
  const [ref, isVisible] = useScrollReveal();

  const customStyle = {
    ...style,
    transitionDelay: `${delay}ms`,
    "--reveal-distance": distance
  };

  return (
    <div
      ref={ref}
      className={`reveal-box reveal-${direction} ${isVisible ? "is-visible" : ""} ${className}`}
      style={customStyle}
      {...props}
    >
      {children}
    </div>
  );
}
