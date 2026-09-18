'use client';

import { SonarGrid } from './sonar-grid';

function DeployPulseLogo() {
  return <span className="deploy-pulse-logo" aria-label="Deploy Pulse"><span className="deploy-pulse-wordmark"><span>DEPLOY</span><strong>PULSE</strong></span></span>;
}

type AnimationPageProps = {
  primaryHref?: string;
  secondaryHref?: string;
  primaryLabel?: string;
};

export function AsciiAnimationBackground({ className = "" }: { className?: string }) {
  return (
    <SonarGrid
      aria-hidden="true"
      interactive={false}
      pingEvery={2.8}
      spacing={28}
      baseOpacity={0.16}
      speed={220}
      ringWidth={100}
      color="#f4f4f5"
      className={className}
    />
  );
}

export default function AnimationPage({
  primaryHref = '/signup',
  secondaryHref = '#insights',
  primaryLabel = 'Get started',
}: AnimationPageProps) {
  return (
    <main className="relative min-h-screen overflow-hidden bg-black">
      <AsciiAnimationBackground className="pointer-events-none absolute inset-0 z-0 h-full w-full opacity-[0.24]" />

      <div className="absolute left-0 right-0 top-0 z-20">
        <div className="container mx-auto flex items-center justify-between px-4 py-3 lg:px-8 lg:py-4">
          <div className="flex items-center gap-2 lg:gap-4">
            <DeployPulseLogo />
            <span className="font-mono text-[8px] text-white/60 lg:text-[10px]">EST. 2025</span>
          </div>

          <div className="hidden items-center gap-4 lg:flex">
            <nav className="flex items-center gap-4 font-mono text-[10px] text-white/60" aria-label="Primary navigation">
              <a className="transition-colors hover:text-white" href="#insights">Features</a>
              <a className="transition-colors hover:text-white" href="#workflow">Architecture</a>
              <a className="transition-colors hover:text-white" href="#security">Security</a>
            </nav>
            <a className="font-mono text-[10px] text-white/60 transition-colors hover:text-white" href="/login">Sign in</a>
            <a className="rounded-md bg-white/10 px-3 py-1.5 font-mono text-[10px] text-white transition-colors hover:bg-white/15 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-white" href={primaryHref}>{primaryLabel}</a>
          </div>
        </div>
      </div>

      <div className="relative z-10 flex min-h-screen items-center justify-center px-6 pt-16 text-center lg:pt-0" style={{ marginTop: '5vh' }}>
        <div className="w-full lg:px-16">
          <div className="mx-auto max-w-4xl">
            <h1 className="mb-5 font-mono text-2xl font-bold leading-tight tracking-wider text-white lg:mb-6 lg:text-5xl" style={{ letterSpacing: '0.1em' }}>
              RELEASES IN VIEW
            </h1>

            <p className="mx-auto mb-5 max-w-3xl font-mono text-xs leading-relaxed text-gray-300 opacity-80 lg:mb-6 lg:text-base">
              Signed webhooks, normalized deployment events, and failure context in one quiet control room. Keep production releases visible from commit to recovery.
            </p>

            <div className="flex flex-col justify-center gap-3 lg:flex-row lg:gap-4">
              <a className="rounded-md bg-white px-5 py-2 text-center font-mono text-xs text-black transition-colors duration-200 hover:bg-white/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-white lg:px-6 lg:py-2.5 lg:text-sm" href={primaryHref}>
                {primaryLabel.toUpperCase()}
              </a>

              <a className="rounded-md bg-white/10 px-5 py-2 text-center font-mono text-xs text-white transition-colors duration-200 hover:bg-white/15 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-white lg:px-6 lg:py-2.5 lg:text-sm" href={secondaryHref}>
                SEE HOW IT WORKS
              </a>
            </div>

            <div className="mt-6 hidden opacity-40 lg:block">
              <span className="font-mono text-[9px] text-white">DEPLOYPULSE.PROTOCOL</span>
            </div>
          </div>
        </div>
      </div>

      <div className="absolute bottom-[5vh] left-0 right-0 z-20">
        <div className="container mx-auto flex items-center justify-between px-4 py-2 lg:px-8 lg:py-3">
          <div className="flex items-center gap-3 font-mono text-[8px] text-white/50 lg:gap-6 lg:text-[9px]">
            <span className="hidden lg:inline">SYSTEM.ACTIVE</span>
            <span className="lg:hidden">SYS.ACT</span>
            <span>INGESTION.ACTIVE</span>
          </div>

          <div className="flex items-center gap-2 font-mono text-[8px] text-white/50 lg:gap-4 lg:text-[9px]">
            <span className="hidden lg:inline">◐ MONITORING</span>
            <span className="hidden lg:inline">EVENTS: LIVE</span>
          </div>
        </div>
      </div>
    </main>
  );
}
