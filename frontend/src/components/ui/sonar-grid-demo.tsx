"use client"

import { ArrowRight } from "lucide-react"
import { motion, useReducedMotion } from "motion/react"
import { SonarGrid } from "./sonar-grid"

const settings = {
  ringWidth: 96,
  speed: 250,
  amplitude: 2.1,
  pingEvery: 2.6,
  interactive: true,
  spacing: 26,
  baseOpacity: 0.2,
  color: "#f4f4f5",
  eyebrow: "DEPLOYMENT EVENT STREAM",
  headline: "RELEASE SIGNALS, WITHOUT NOISE.",
  subline: "Every verified webhook becomes a clear operational signal—from commit to recovery.",
}

export default function SonarGridDemo(props: Partial<typeof settings>) {
  const s = { ...settings, ...props }
  const reduce = useReducedMotion()
  const enter = (delay: number) =>
    reduce
      ? {}
      : {
          initial: { opacity: 0, y: 14, filter: "blur(6px)" },
          animate: { opacity: 1, y: 0, filter: "blur(0px)" },
          transition: { duration: 0.6, delay, ease: [0.22, 1, 0.36, 1] as const },
        }

  return (
    <SonarGrid
      id="sonar-grid-demo"
      ringWidth={s.ringWidth}
      speed={s.speed}
      amplitude={s.amplitude}
      pingEvery={s.pingEvery}
      interactive={s.interactive}
      spacing={s.spacing}
      baseOpacity={s.baseOpacity}
      color={s.color}
      pingArea={[0.18, 0.16, 0.82, 0.84]}
      className="flex min-h-[max(560px,100svh)] w-full flex-col bg-black text-white"
    >
      <div aria-hidden="true" className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(ellipse_34%_30%_at_50%_50%,rgba(255,255,255,0.08)_0%,transparent_100%)]" />
      <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col items-center justify-center px-8 py-24 text-center">
        <div className="flex max-w-3xl flex-col items-center">
          <motion.p {...enter(0)} className="mb-5 inline-flex items-center gap-2 rounded-full border border-white/30 px-3 py-1 text-xs font-medium tracking-[0.18em] text-white/70">
            <span aria-hidden="true" className="size-1.5 rounded-full bg-white" />
            {s.eyebrow}
          </motion.p>
          <motion.h1 {...enter(0.08)} className="text-balance font-mono text-4xl font-semibold tracking-tight text-white sm:text-6xl md:text-7xl">
            {s.headline}
          </motion.h1>
          <motion.p {...enter(0.16)} className="mt-6 max-w-xl text-pretty font-mono text-base leading-relaxed text-white/60 sm:text-lg">
            {s.subline}
          </motion.p>
          <motion.div {...enter(0.24)} className="mt-9 flex flex-wrap items-center justify-center gap-3">
            <a href="/app" className="group inline-flex h-11 items-center gap-2 border border-white bg-white px-6 text-sm font-medium text-black transition hover:bg-transparent hover:text-white">
              Open release console
              <ArrowRight aria-hidden="true" className="size-4 transition-transform duration-200 group-hover:translate-x-0.5" />
            </a>
            <a href="/#workflow" className="inline-flex h-11 items-center border border-white/40 px-6 text-sm font-medium text-white transition hover:border-white hover:bg-white hover:text-black">
              View workflow
            </a>
          </motion.div>
        </div>
      </div>
    </SonarGrid>
  )
}
