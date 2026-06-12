"use client"

import {motion} from "motion/react"
import {RegisterForm} from "@/components/auth/register-form"
import {AuroraBackground} from "@/components/ui/aurora-background"

export function RegisterPage() {
  return (
    <AuroraBackground>
      <motion.div
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{
          delay: 0.3,
          duration: 0.8,
          ease: "easeInOut",
        }}
        className="relative z-10 w-full max-w-sm px-4"
      >
        <RegisterForm />
      </motion.div>
    </AuroraBackground>
  )
}
