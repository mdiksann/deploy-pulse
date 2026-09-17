import React, { useEffect, useRef, useState } from "react";

export function RadarGlobeCanvas({ activeHotspot = "spot-1" }) {
  const canvasRef = useRef(null);
  const [selectedSpot, setSelectedSpot] = useState(activeHotspot);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    let ctx = null;
    try {
      if (typeof canvas.getContext === "function") {
        ctx = canvas.getContext("2d");
      }
    } catch {
      ctx = null;
    }

    if (!ctx) return; // Guard for headless/JSDOM testing

    let animationFrameId;
    let rotation = 0;
    let tilt = 0.35;

    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      const dpr = window.devicePixelRatio || 1;
      canvas.width = (rect.width || 280) * dpr;
      canvas.height = (rect.height || 180) * dpr;
      if (ctx.scale) {
        ctx.scale(dpr, dpr);
      }
    };

    resize();
    window.addEventListener("resize", resize);

    const hotspots = [
      { id: "spot-1", name: "Spot 1", label: "US-East", lat: 0.3, lon: 0.8, color: "#d8eee4" },
      { id: "spot-2", name: "Spot 2", label: "EU-West", lat: 0.6, lon: 2.4, color: "#a5d6c2" },
      { id: "spot-3", name: "Spot 3", label: "AP-East", lat: -0.2, lon: 4.2, color: "#8fc9b0" }
    ];

    const render = () => {
      const rect = canvas.getBoundingClientRect();
      const width = rect.width || 280;
      const height = rect.height || 180;
      if (width === 0 || height === 0) return;

      ctx.clearRect(0, 0, width, height);

      const cx = width / 2;
      const cy = height / 2;
      const radius = Math.min(width, height) * 0.42;

      rotation += 0.008;

      // 1. Ambient outer aura ring (soft sage mist)
      const auraGradient = ctx.createRadialGradient(cx, cy, radius * 0.4, cx, cy, radius * 1.3);
      auraGradient.addColorStop(0, "rgba(180, 220, 205, 0.12)");
      auraGradient.addColorStop(0.6, "rgba(145, 195, 175, 0.04)");
      auraGradient.addColorStop(1, "rgba(0, 0, 0, 0)");
      ctx.fillStyle = auraGradient;
      ctx.beginPath();
      ctx.arc(cx, cy, radius * 1.3, 0, Math.PI * 2);
      ctx.fill();

      // 2. Wireframe latitude rings
      ctx.lineWidth = 1;
      const latCount = 5;
      for (let i = -latCount; i <= latCount; i++) {
        const latRatio = i / (latCount + 1);
        const yOffset = latRatio * radius * Math.cos(tilt);
        const rLat = radius * Math.sqrt(Math.max(0, 1 - latRatio * latRatio));

        ctx.strokeStyle = "rgba(100, 130, 120, 0.28)";
        ctx.beginPath();
        ctx.ellipse(cx, cy + yOffset, rLat, rLat * Math.sin(tilt), 0, 0, Math.PI * 2);
        ctx.stroke();
      }

      // 3. Rotating longitude arcs
      const lonCount = 8;
      for (let i = 0; i < lonCount; i++) {
        const lonAngle = rotation + (i * Math.PI) / lonCount;
        const xOffset = Math.sin(lonAngle);
        const zOffset = Math.cos(lonAngle);

        ctx.strokeStyle = zOffset > 0 ? "rgba(180, 220, 205, 0.35)" : "rgba(80, 105, 95, 0.18)";
        ctx.beginPath();
        ctx.ellipse(cx, cy, Math.abs(xOffset) * radius, radius, 0, 0, Math.PI * 2);
        ctx.stroke();
      }

      // 4. Outer orbital track
      ctx.strokeStyle = "rgba(165, 215, 195, 0.45)";
      ctx.setLineDash([4, 6]);
      ctx.beginPath();
      ctx.ellipse(cx, cy, radius * 1.12, radius * 0.48, -0.2, 0, Math.PI * 2);
      ctx.stroke();
      ctx.setLineDash([]);

      // 5. Render Hotspots
      hotspots.forEach((spot) => {
        const spotLon = spot.lon + rotation;
        const x3d = Math.cos(spot.lat) * Math.sin(spotLon);
        const y3d = Math.sin(spot.lat);
        const z3d = Math.cos(spot.lat) * Math.cos(spotLon);

        if (z3d > -0.2) {
          const screenX = cx + x3d * radius;
          const screenY = cy - y3d * radius * Math.cos(tilt) + (z3d * radius * Math.sin(tilt) * 0.3);
          const isSelected = selectedSpot === spot.id;

          const pulse = (Date.now() % 2000) / 2000;
          ctx.strokeStyle = isSelected ? "#ffffff" : spot.color;
          ctx.lineWidth = 1;
          ctx.beginPath();
          ctx.arc(screenX, screenY, 4 + pulse * 10, 0, Math.PI * 2);
          ctx.stroke();

          ctx.fillStyle = isSelected ? "#ffffff" : spot.color;
          ctx.beginPath();
          ctx.arc(screenX, screenY, isSelected ? 3.5 : 2.5, 0, Math.PI * 2);
          ctx.fill();

          ctx.font = "9px 'JetBrains Mono', monospace";
          ctx.fillStyle = isSelected ? "#ffffff" : "rgba(226, 232, 240, 0.75)";
          ctx.fillText(spot.name, screenX + 6, screenY - 4);
        }
      });

      animationFrameId = requestAnimationFrame(render);
    };

    animationFrameId = requestAnimationFrame(render);

    return () => {
      window.removeEventListener("resize", resize);
      if (animationFrameId) cancelAnimationFrame(animationFrameId);
    };
  }, [selectedSpot]);

  return (
    <div className="radar-globe-wrapper">
      <canvas ref={canvasRef} className="radar-globe-canvas" />
      <div className="radar-spots-chips">
        <button
          type="button"
          className={`spot-chip ${selectedSpot === "spot-1" ? "active" : ""}`}
          onClick={() => setSelectedSpot("spot-1")}
        >
          <i /> Spot 1
        </button>
        <button
          type="button"
          className={`spot-chip ${selectedSpot === "spot-2" ? "active" : ""}`}
          onClick={() => setSelectedSpot("spot-2")}
        >
          <i /> Spot 2
        </button>
        <button
          type="button"
          className={`spot-chip ${selectedSpot === "spot-3" ? "active" : ""}`}
          onClick={() => setSelectedSpot("spot-3")}
        >
          <i /> Spot 3
        </button>
      </div>
    </div>
  );
}
