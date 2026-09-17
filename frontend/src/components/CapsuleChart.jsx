import React, { useState } from "react";

const DEFAULT_BARS = [
  { id: "b1", height: 72, segments: 4, value: "98.4%", label: "Lint & Types", color: "cyan" },
  { id: "b2", height: 94, segments: 5, value: "99.9%", label: "Unit Tests", color: "aqua" },
  { id: "b3", height: 62, segments: 3, value: "94.2%", label: "Container Build", color: "sky" },
  { id: "b4", height: 86, segments: 4, value: "97.8%", label: "Canary Verify", color: "white" },
  { id: "b5", height: 54, segments: 3, value: "91.5%", label: "DB Migration", color: "cyan" },
  { id: "b6", height: 82, segments: 5, value: "99.2%", label: "Health Check", color: "electric" }
];

export function CapsuleChart({ data = DEFAULT_BARS }) {
  const [hoveredBar, setHoveredBar] = useState(null);

  return (
    <div className="capsule-chart-container" role="region" aria-label="Pipeline Health Capsule Chart">
      <div className="capsule-bars-grid">
        {data.map((bar) => {
          const isHovered = hoveredBar?.id === bar.id;
          return (
            <div
              key={bar.id}
              className={`capsule-bar-col ${bar.color} ${isHovered ? "is-hovered" : ""}`}
              onMouseEnter={() => setHoveredBar(bar)}
              onMouseLeave={() => setHoveredBar(null)}
              tabIndex={0}
              role="figure"
              aria-label={`${bar.label}: ${bar.value}`}
            >
              <div className="capsule-track">
                <div
                  className="capsule-fill"
                  style={{ height: `${bar.height}%` }}
                >
                  <span className="capsule-cap" />
                  <div className="capsule-segments">
                    {Array.from({ length: bar.segments }).map((_, i) => (
                      <span key={i} className="capsule-segment-ring" />
                    ))}
                  </div>
                </div>
              </div>
              <span className="capsule-base-dot" />
            </div>
          );
        })}
      </div>

      <div className="capsule-hud-detail">
        {hoveredBar ? (
          <div className="capsule-tooltip-active">
            <strong>{hoveredBar.label}</strong>
            <span>Check Velocity: <em>{hoveredBar.value}</em></span>
          </div>
        ) : (
          <div className="capsule-tooltip-idle">
            <span className="pulse-indicator" />
            <span>Hover bars for pipeline check velocity</span>
          </div>
        )}
      </div>
    </div>
  );
}
