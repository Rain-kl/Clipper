"use client"

import { useCallback, useEffect, useState } from "react"
import { motion, AnimatePresence } from "motion/react"
import { useRouter, useSearchParams } from "next/navigation"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { LoginForm } from "@/components/auth/login-form"
import { Check } from "lucide-react"

import { AuroraBackground } from "@/components/ui/aurora-background"
import services from "@/lib/services"
import type { ApiResponse } from "@/lib/services/core/types"


/**
 * 登录页面组件
 * 显示登录表单和登录按钮
 * 
 * @example
 * ```tsx
 * <LoginPage />
 * ```
 * @returns {React.ReactNode} 登录页面组件
 */
export function LoginPage() {
  const router = useRouter()
  const searchParams = useSearchParams()

  /* 处理OAuth回调 */
  const [isProcessingCallback, setIsProcessingCallback] = useState(() => {
    const state = searchParams.get('state')
    const code = searchParams.get('code')
    return !!(state && code)
  })
  const [isCheckingSession, setIsCheckingSession] = useState(() => !searchParams.get('state') || !searchParams.get('code'))

  const [loginSuccess, setLoginSuccess] = useState(false)

  const resolveRedirectTarget = useCallback(() => {
    const callbackUrl = searchParams.get('callbackUrl')
    const storedRedirect = sessionStorage.getItem('redirect_after_login')
    const target = callbackUrl || storedRedirect || '/home'

    if (storedRedirect) {
      sessionStorage.removeItem('redirect_after_login')
    }

    return target
  }, [searchParams])

  /* 登录页兜底：已登录用户直接跳转 */
  useEffect(() => {
    const state = searchParams.get('state')
    const code = searchParams.get('code')

    if (state && code) {
      setIsCheckingSession(false)
      return
    }

    let cancelled = false

    const checkExistingSession = async () => {
      setIsCheckingSession(true)

      try {
        const response = await fetch('/api/v1/oauth/user-info', {
          credentials: 'include',
          cache: 'no-store',
        })

        if (cancelled) return

        if (response.ok) {
          await response.json() as ApiResponse
          router.replace(resolveRedirectTarget())
          return
        }

        if (response.status !== 401) {
          console.error('Session probe failed:', response.status)
        }
      } catch (error) {
        if (!cancelled) {
          console.error('Session probe error:', error)
        }
      } finally {
        if (!cancelled) {
          setIsCheckingSession(false)
        }
      }
    }

    checkExistingSession()

    return () => {
      cancelled = true
    }
  }, [router, searchParams, resolveRedirectTarget])

  /* 回调逻辑 */
  useEffect(() => {
    const handleOAuthCallback = async () => {
      const state = searchParams.get('state')
      const code = searchParams.get('code')

      if (state && code) {
        setIsProcessingCallback(true)
        try {
          await services.auth.handleCallback({ state, code })
          setLoginSuccess(true)
          toast.success("登录成功")

          setTimeout(() => {
            router.replace(resolveRedirectTarget())
          }, 1500)
        } catch (error) {
          console.error('OAuth callback error:', error)
          toast.error(error instanceof Error ? error.message : "登录失败，请重试")
          setIsProcessingCallback(false)
          router.replace('/login')
        }
      }
    }
    handleOAuthCallback()
  }, [searchParams, router, resolveRedirectTarget])

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
        <div className="text-center mb-8 space-y-2">
          <h1 className="text-3xl font-bold tracking-tight text-foreground">
            LINUX DO <span className="font-serif italic text-primary">Credit</span>
          </h1>
          <p className="text-sm text-muted-foreground font-light">
            简单、安全，专为社区设计
          </p>
        </div>

        <AnimatePresence mode="wait">
          {isProcessingCallback || isCheckingSession ? (
            <motion.div
              key={isProcessingCallback ? "processing" : "session-check"}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="w-full"
            >
              {isCheckingSession ? (
                <div className="flex flex-col items-center justify-center space-y-4 py-2">
                  <div className="relative">
                    <Spinner className="w-8 h-8 text-blue-600" />
                  </div>
                  <div className="text-center space-y-2">
                    <h3 className="font-semibold tracking-tight text-foreground">正在检查登录状态</h3>
                    <p className="text-xs text-muted-foreground">请稍候，我们正在确认当前会话...</p>
                  </div>
                </div>
              ) : loginSuccess ? (
                <div className="flex flex-col items-center justify-center space-y-4 py-2">
                  <motion.div
                    initial={{ scale: 0.5, opacity: 0 }}
                    animate={{ scale: 1, opacity: 1 }}
                    transition={{ type: "spring", stiffness: 300, damping: 20 }}
                    className="w-8 h-8 rounded-full bg-green-500/10 flex items-center justify-center text-green-500 ring-1 ring-green-500/20"
                  >
                    <Check className="w-6 h-6" strokeWidth={3} />
                  </motion.div>
                  <div className="text-center space-y-2">
                    <h3 className="font-semibold tracking-tight text-foreground">登录成功</h3>
                    <p className="text-xs text-muted-foreground">正在跳转至控制台...</p>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center space-y-4 py-2">
                  <div className="relative">
                    <Spinner className="w-8 h-8 text-blue-600" />
                  </div>
                  <div className="text-center space-y-2">
                    <h3 className="font-semibold tracking-tight text-foreground">正在验证凭据</h3>
                    <p className="text-xs text-muted-foreground">请稍候，我们正在为您建立安全会话...</p>
                  </div>
                </div>
              )}
            </motion.div>
          ) : (
            <motion.div
              key="login-form-wrapper"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.4 }}
              className="w-full"
            >
              <LoginForm />
            </motion.div>
          )}
        </AnimatePresence>

        <div className="mt-8 text-center text-xs text-muted-foreground">
          &copy; {new Date().getFullYear()} LINUX DO Credit. 版权所有
        </div>
      </motion.div>
    </AuroraBackground>
  )
}
