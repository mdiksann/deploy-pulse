import React, { useRef, useState } from "react";

export function TiltCard({
  children,
  className = "",
  maxTilt = 6,
  glare = true,
  style = {},
  onClick,
  ...props
}) {
  const cardRef = useRef(null);
  const [transform, setTransform] = useState("perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)");
  const [glarePos, setGlarePos] = useState({ x: 50, y: 50, opacity: 0 });

  const handleMouseMove = (e) => {
    const card = cardRef.current;
    if (!card) return;

    const rect = card.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const width = rect.width;
    const height = rect.height;

    const normX = (x / width - 0.5) * 2; // -1 to 1
    const normY = (y / height - 0.5) * 2; // -1 to 1

    const rotX = -normY * maxTilt;
    const rotY = normX * maxTilt;

    setTransform(`perspective(1000px) rotateX(${rotX.toFixed(2)}deg) rotateY(${rotY.toFixed(2)}deg) scale3d(1.015, 1.015, 1.015)`);
    setGlarePos({
      x: ((x / width) * 100).toFixed(1),
      y: ((y / height) * 100).toFixed(1),
      opacity: 0.18
    });
  };

  const handleMouseLeave = () => {
    setTransform("perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)");
    setGlarePos((prev) => ({ ...prev, opacity: 0 }));
  };

  return (
    <div
      ref={cardRef}
      className={`tilt-card-wrapper ${className}`}
      style={{
        ...style,
        transform,
        transition: "transform 0.18s cubic-bezier(0.2, 0.8, 0.4, 1)"
      }}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      onClick={onClick}
      {...props}
    >
      {glare && (
        <div
          className="tilt-glare-overlay"
          style={{
            background: `radial-gradient(circle at ${glarePos.x}% ${glarePos.y}%, rgba(0, 245, 155, ${glarePos.opacity}), transparent 60%)`
          }}
          aria-hidden="true"
        />
      )}
      {children}
    </div>
  );
}
