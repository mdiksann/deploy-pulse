import React, { useEffect, useRef } from "react";

export function OscilloscopeCanvas({ activePulse = false, intensity = 1, tone = "cyan" }) {
  const canvasRef = useRef(null);

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
    if (!ctx) return; // Guard for JSDOM or environments without full Canvas support

    let animationFrameId;
    let phase = 0;
    let pulseDecay = 0;

    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      const dpr = window.devicePixelRatio || 1;
      canvas.width = (rect.width || 300) * dpr;
      canvas.height = (rect.height || 90) * dpr;
      if (ctx.scale) {
        ctx.scale(dpr, dpr);
      }
    };

    resize();
    window.addEventListener("resize", resize);

    const render = () => {
      const rect = canvas.getBoundingClientRect();
      const width = rect.width || 300;
      const height = rect.height || 90;
      if (width === 0 || height === 0) return;

      // Dark CRT scan canvas
      ctx.clearRect(0, 0, width, height);

      // Background subtle grid
      ctx.strokeStyle = "rgba(14, 165, 233, 0.12)";
      ctx.lineWidth = 1;
      ctx.beginPath();
      const gridStep = 20;
      for (let x = 0; x < width; x += gridStep) {
        ctx.moveTo(x, 0);
        ctx.lineTo(x, height);
      }
      for (let y = 0; y < height; y += gridStep) {
        ctx.moveTo(0, y);
        ctx.lineTo(width, y);
      }
      ctx.stroke();

      // Center crosshair axis
      ctx.strokeStyle = "rgba(56, 189, 248, 0.25)";
      ctx.beginPath();
      ctx.moveTo(0, height / 2);
      ctx.lineTo(width, height / 2);
      ctx.stroke();

      // Waveform calculation
      const primaryColor = tone === "cyan" ? "#38bdf8" : tone === "emerald" ? "#10b981" : tone === "crimson" ? "#f43f5e" : "#06b6d4";
      const glowColor = tone === "cyan" ? "rgba(56, 189, 248, 0.45)" : tone === "emerald" ? "rgba(16, 185, 129, 0.4)" : "rgba(244, 63, 94, 0.4)";

      phase += 0.08;
      if (activePulse) {
        pulseDecay = Math.min(pulseDecay + 0.15, 1.2);
      } else {
        pulseDecay = Math.max(pulseDecay - 0.03, 0);
      }

      ctx.save();
      ctx.beginPath();
      ctx.strokeStyle = primaryColor;
      ctx.shadowColor = glowColor;
      ctx.shadowBlur = 10;
      ctx.lineWidth = 2;

      const midY = height / 2;
      const points = [];

      for (let x = 0; x <= width; x += 3) {
        const normX = x / width;
        // Base carrier wave
        let y = Math.sin(normX * 18 - phase) * (4 * intensity);
        // Harmonic noise
        y += Math.sin(normX * 42 + phase * 1.5) * 2;
        // Pulse burst
        if (pulseDecay > 0.01) {
          const packetCenter = 0.5 + Math.sin(phase * 0.5) * 0.3;
          const dist = Math.abs(normX - packetCenter);
          const pulseShape = Math.exp(-Math.pow(dist / 0.15, 2));
          y += Math.sin(normX * 80 - phase * 4) * (24 * pulseDecay * pulseShape);
        }

        const plotY = midY + y;
        points.push({ x, y: plotY });
        if (x === 0) {
          ctx.moveTo(x, plotY);
        } else {
          ctx.lineTo(x, plotY);
        }
      }
      ctx.stroke();
      ctx.restore();

      // Laser scanning point
      const sweepX = ((phase * 40) % (width + 40)) - 20;
      if (sweepX >= 0 && sweepX <= width) {
        ctx.fillStyle = primaryColor;
        ctx.beginPath();
        ctx.arc(sweepX, midY, 2.5, 0, Math.PI * 2);
        ctx.fill();
      }

      animationFrameId = requestAnimationFrame(render);
    };

    animationFrameId = requestAnimationFrame(render);

    return () => {
      window.removeEventListener("resize", resize);
      if (animationFrameId) {
        cancelAnimationFrame(animationFrameId);
      }
    };
  }, [activePulse, intensity, tone]);

  return (
    <div className="oscilloscope-wrapper" aria-hidden="true">
      <canvas ref={canvasRef} className="oscilloscope-canvas" />
      <div className="oscilloscope-hud">
        <span className="hud-badge"><i />FREQ: 44.1 kHz</span>
        <span className="hud-badge"><i />SIG: {activePulse ? "BURST // XADD" : "IDLE // LISTENING"}</span>
      </div>
    </div>
  );
}
