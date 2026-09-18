import "@testing-library/jest-dom";

// jsdom does not implement canvas rendering. Returning null lets canvas-based
// decorative components safely no-op during DOM-only tests without noisy logs.
if (typeof HTMLCanvasElement !== "undefined") {
  HTMLCanvasElement.prototype.getContext = () => null;
}
