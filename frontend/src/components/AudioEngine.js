// Subtle Web Audio API sound synthesizer for tactile UI micro-interactions
let audioCtx = null;
let soundEnabled = false;

export function isAudioSupported() {
  return typeof window !== "undefined" && (window.AudioContext || window.webkitAudioContext);
}

export function toggleAudio(enable) {
  if (enable === undefined) {
    soundEnabled = !soundEnabled;
  } else {
    soundEnabled = Boolean(enable);
  }
  if (soundEnabled && !audioCtx && typeof window !== "undefined") {
    const AudioContextClass = window.AudioContext || window.webkitAudioContext;
    if (AudioContextClass) {
      audioCtx = new AudioContextClass();
    }
  }
  if (soundEnabled && audioCtx && audioCtx.state === "suspended") {
    audioCtx.resume().catch(() => {});
  }
  return soundEnabled;
}

export function getAudioState() {
  return soundEnabled;
}

export function playTactileSound(type = "tick") {
  if (!soundEnabled || !audioCtx) return;
  try {
    if (audioCtx.state === "suspended") {
      audioCtx.resume();
    }
    const t = audioCtx.currentTime;
    const osc = audioCtx.createOscillator();
    const gain = audioCtx.createGain();
    osc.connect(gain);
    gain.connect(audioCtx.destination);

    if (type === "tick") {
      osc.type = "sine";
      osc.frequency.setValueAtTime(800, t);
      osc.frequency.exponentialRampToValueAtTime(300, t + 0.03);
      gain.gain.setValueAtTime(0.04, t);
      gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.03);
      osc.start(t);
      osc.stop(t + 0.035);
    } else if (type === "pulse") {
      osc.type = "triangle";
      osc.frequency.setValueAtTime(440, t);
      osc.frequency.exponentialRampToValueAtTime(880, t + 0.06);
      gain.gain.setValueAtTime(0.05, t);
      gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.06);
      osc.start(t);
      osc.stop(t + 0.065);
    } else if (type === "success") {
      osc.type = "sine";
      osc.frequency.setValueAtTime(523.25, t);
      osc.frequency.setValueAtTime(659.25, t + 0.04);
      osc.frequency.setValueAtTime(783.99, t + 0.08);
      gain.gain.setValueAtTime(0.05, t);
      gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.15);
      osc.start(t);
      osc.stop(t + 0.16);
    } else if (type === "error") {
      osc.type = "sawtooth";
      osc.frequency.setValueAtTime(220, t);
      osc.frequency.linearRampToValueAtTime(140, t + 0.12);
      gain.gain.setValueAtTime(0.06, t);
      gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.14);
      osc.start(t);
      osc.stop(t + 0.15);
    }
  } catch (err) {
    // Graceful fallback
  }
}
