"use client"

import {useEffect, useMemo, useRef, useState} from "react"
import {useMutation, useQuery} from "@tanstack/react-query"
import {useRouter, useSearchParams} from "next/navigation"
import {KeyRound} from "lucide-react"
import {toast} from "sonner"
import Link from "next/link"

import {useAuth} from "@/components/providers/auth-provider"
import {Button} from "@/components/ui/button"
import {Input} from "@/components/ui/input"
import {Label} from "@/components/ui/label"
import {Separator} from "@/components/ui/separator"
import {Spinner} from "@/components/ui/spinner"
import {Card, CardContent} from "@/components/ui/card"
import {CapWidget} from "@/components/auth/cap-widget"
import {OTPForm} from "./otp-form"
import services from "@/lib/services"
import type {LoginRequest} from "@/lib/services/auth/types"

function getRedirectTarget(searchParams: ReturnType<typeof useSearchParams>) {
  const callbackUrl = searchParams.get("callbackUrl")
  const storedRedirect = sessionStorage.getItem("redirect_after_login")
  const target = callbackUrl || storedRedirect || "/home"

  if (storedRedirect) {
    sessionStorage.removeItem("redirect_after_login")
  }

  return target
}

function persistRedirectTarget(searchParams: ReturnType<typeof useSearchParams>) {
  const callbackUrl = searchParams.get("callbackUrl")
  if (callbackUrl) {
    sessionStorage.setItem("redirect_after_login", callbackUrl)
  }
}

function configBool(value: string | undefined, fallback: boolean) {
  if (value === undefined) return fallback
  return value === "true"
}

export function LoginForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { setUser } = useAuth()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [code, setCode] = useState("")
  const [showLoginCodeInput, setShowLoginCodeInput] = useState(false)
  const [loginCooldown, setLoginCooldown] = useState(0)
  const [errorMessage, setErrorMessage] = useState("")
  const [loginCodeTip, setLoginCodeTip] = useState<React.ReactNode>(null)

  useEffect(() => {
    if (loginCooldown > 0) {
      const timer = setTimeout(() => setLoginCooldown(loginCooldown - 1), 1000)
      return () => clearTimeout(timer)
    }
  }, [loginCooldown])

  // Cap token management — ref to hold latest token without triggering re-render
  const capTokenRef = useRef<string | null>(null)
  const [capReady, setCapReady] = useState(false)
  const [capError, setCapError] = useState(false)
  const [capResetKey, setCapResetKey] = useState(0)

  const publicConfigQuery = useQuery({
    queryKey: ["public-config"],
    queryFn: () => services.config.getPublicConfig(),
  })

  const authSourcesQuery = useQuery({
    queryKey: ["auth-sources"],
    queryFn: () => services.auth.getAuthSources(),
    enabled: configBool(publicConfigQuery.data?.oidc_login_enabled, true),
  })

  const redirectTarget = useMemo(
    () => getRedirectTarget(searchParams),
    [searchParams],
  )

  const capEnabled = configBool(publicConfigQuery.data?.cap_login_enabled, false)
  const capAutoSolve = configBool(publicConfigQuery.data?.cap_auto_solve, true)

  const loginMutation = useMutation({
    mutationFn: (req: LoginRequest) => {
      const headers: Record<string, string> = {}
      if (capEnabled && capTokenRef.current) {
        headers["X-Cap-Token"] = capTokenRef.current
        // Consume the token — next login attempt will need a new one
        capTokenRef.current = null
        setCapReady(false)
      }
      return services.auth.login(req, Object.keys(headers).length ? headers : undefined)
    },
    onSuccess: (user) => {
      setUser(user)
      router.replace(redirectTarget)
      toast.success("登录成功")
    },
    onError: (error: Error) => {
      const errorMsg = error.message || ""
      if (errorMsg.startsWith("need_email_code:")) {
        const emailMasked = errorMsg.substring("need_email_code:".length)
        setLoginCodeTip(
          <>
            已向您的安全邮箱 <span className="font-medium text-foreground">{emailMasked}</span> 发送了登录验证码。
          </>
        )
        setShowLoginCodeInput(true)
        setLoginCooldown(60)
        toast.success("登录验证码已发送至您的邮箱，请注意查收")
        if (capEnabled) {
          capTokenRef.current = null
          setCapReady(false)
          setCapResetKey((key) => key + 1)
        }
        return
      }

      if (errorMsg.startsWith("smtp_invalid:")) {
        const tip = errorMsg.substring("smtp_invalid:".length)
        setLoginCodeTip(
          <span className="text-amber-500 font-medium">{tip}</span>
        )
        setShowLoginCodeInput(true)
        setLoginCooldown(0)
        toast.warning(tip)
        if (capEnabled) {
          capTokenRef.current = null
          setCapReady(false)
          setCapResetKey((key) => key + 1)
        }
        return
      }

      setErrorMessage(errorMsg || "登录失败，请重试")
      if (capEnabled) {
        capTokenRef.current = null
        setCapReady(false)
        setCapResetKey((key) => key + 1)
      }
    },
  })

  const handlePasswordLogin = () => {
    setErrorMessage("")
    const trimmedUsername = username.trim()
    if (!trimmedUsername || !password) {
      toast.error("邮箱/用户名或密码未填写完整", {
        description: "请先输入邮箱/用户名和密码后再登录",
      })
      return
    }
    if (capEnabled && !capReady) {
      toast.error(
        capAutoSolve
          ? "人机验证尚未完成，请稍候…"
          : "请先点击「开始验证」完成人机验证",
      )
      return
    }
    loginMutation.mutate({
      username: trimmedUsername,
      password,
      code: showLoginCodeInput ? code.trim() : undefined,
    })
  }

  const handleResendLoginCode = () => {
    setCode("")
    loginMutation.mutate({
      username: username.trim(),
      password,
    })
  }

  const handleOAuthLogin = async (sourceName: string) => {
    try {
      setErrorMessage("")
      persistRedirectTarget(searchParams)
      const { authorize_url } = await services.auth.getAuthorizeUrl(sourceName)
      window.location.href = authorize_url
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "第三方登录失败")
    }
  }

  const handleCapToken = (token: string) => {
    capTokenRef.current = token
    setCapReady(true)
    setCapError(false)
  }

  const handleCapError = () => {
    capTokenRef.current = null
    setCapReady(false)
    setCapError(true)
  }

  const registrationEnabled =
    configBool(publicConfigQuery.data?.registration_enabled, true) &&
    configBool(publicConfigQuery.data?.password_register_enabled, true)

  const passwordLoginEnabled = configBool(publicConfigQuery.data?.password_login_enabled, true)
  const authSources = authSourcesQuery.data ?? []

  const loginDisabled =
    !passwordLoginEnabled ||
    loginMutation.isPending ||
    (capEnabled && capAutoSolve && !capReady && !capError)

  if (publicConfigQuery.isPending) {
    return (
      <Card className="w-full border-border/60 bg-background/80 shadow-2xl backdrop-blur">
        <CardContent className="flex justify-center items-center py-12">
          <Spinner />
        </CardContent>
      </Card>
    )
  }

  if (showLoginCodeInput) {
    return (
      <OTPForm
        code={code}
        setCode={setCode}
        loginCodeTip={loginCodeTip}
        loginCooldown={loginCooldown}
        isPending={loginMutation.isPending}
        onResend={handleResendLoginCode}
        onSubmit={handlePasswordLogin}
      />
    )
  }

  return (
    <Card className="w-full border-border/60 bg-background/80 shadow-2xl backdrop-blur">
      <CardContent className="space-y-5 p-5 sm:p-6">
        <div className="space-y-2 text-center">
          <h2 className="text-xl font-semibold tracking-tight text-foreground">
            账号登录
          </h2>
        </div>

        <div className="space-y-4 pt-2">
          <div className="space-y-3">
            <div className="space-y-3">
              <div className="space-y-1.5">
                <Label htmlFor="username">邮箱/用户名</Label>
                <Input
                  id="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="请输入邮箱或用户名"
                  autoComplete="username"
                  onKeyDown={(e) => e.key === "Enter" && handlePasswordLogin()}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="password">密码</Label>
                <Input
                  id="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  type="password"
                  placeholder="请输入密码"
                  autoComplete="current-password"
                  onKeyDown={(e) => e.key === "Enter" && handlePasswordLogin()}
                />
              </div>

            </div>

            {/* Cap 人机验证 */}
            {capEnabled && (
              <CapWidget
                key={capResetKey}
                onToken={handleCapToken}
                onError={handleCapError}
                autoStart={capAutoSolve}
              />
            )}

            {errorMessage ? (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {errorMessage}
              </div>
            ) : null}

            <Button
              type="button"
              className="w-full"
              variant={"secondary"}
              onClick={handlePasswordLogin}
              disabled={loginDisabled}
            >
              {loginMutation.isPending ? (
                <>
                  <Spinner className="mr-2" />
                  登录中...
                </>
              ) : (
                <>
                  <KeyRound className="mr-2 size-4" />
                  登录
                </>
              )}
            </Button>
          </div>
        </div>

        {authSources.length > 0 ? (
          <>
            <Separator />
            <div className="space-y-3">
              <div className="grid gap-2">
                {authSources.map((source) => (
                  <Button
                    key={source.id}
                    type="button"
                    variant="outline"
                    className="justify-start"
                    onClick={() => void handleOAuthLogin(source.name)}
                  >
                    {source.display_name || source.name} 登录
                  </Button>
                ))}
              </div>
            </div>
          </>
        ) : null}

        {registrationEnabled && (
          <div className="text-center text-xs text-muted-foreground mt-4">
            {"Don't have an account?"}{" "}
            <Link href="/register" className="font-semibold text-indigo-500 hover:text-indigo-600 transition-colors">
              Sign up
            </Link>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
