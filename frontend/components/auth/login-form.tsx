"use client"

import {useMemo, useRef, useState} from "react"
import {useMutation, useQuery} from "@tanstack/react-query"
import {useRouter, useSearchParams} from "next/navigation"
import {KeyRound, ShieldCheck, UserPlus} from "lucide-react"
import {toast} from "sonner"

import {useAuth} from "@/components/providers/auth-provider"
import {Button} from "@/components/ui/button"
import {Input} from "@/components/ui/input"
import {Separator} from "@/components/ui/separator"
import {Spinner} from "@/components/ui/spinner"
import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {Card, CardContent} from "@/components/ui/card"
import {CapWidget} from "@/components/auth/cap-widget"
import services from "@/lib/services"
import type {LoginRequest, RegisterRequest} from "@/lib/services/auth/types"

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

export function LoginForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { setUser } = useAuth()
  const [mode, setMode] = useState<"login" | "register">("login")
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [nickname, setNickname] = useState("")
  const [errorMessage, setErrorMessage] = useState("")

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
    enabled: publicConfigQuery.data?.oidc_login_enabled ?? true,
  })

  const redirectTarget = useMemo(
    () => getRedirectTarget(searchParams),
    [searchParams],
  )

  const capEnabled = publicConfigQuery.data?.cap_login_enabled ?? false
  const capAutoSolve = publicConfigQuery.data?.cap_auto_solve ?? true

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
    },
    onError: (error: Error) => {
      toast.error(error.message || "登录失败，请重试")
      if (capEnabled) {
        capTokenRef.current = null
        setCapReady(false)
        setCapResetKey((key) => key + 1)
      }
    },
  })

  const registerMutation = useMutation({
    mutationFn: (req: RegisterRequest) => services.auth.register(req),
    onSuccess: (user) => {
      setUser(user)
      router.replace(redirectTarget)
    },
    onError: (error: Error) => {
      setErrorMessage(error.message || "注册失败，请重试")
    },
  })

  const handlePasswordLogin = () => {
    setErrorMessage("")
    const trimmedUsername = username.trim()
    if (!trimmedUsername || !password) {
      toast.error("账号或密码未填写完整", {
        description: "请先输入账号和密码后再登录",
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
    })
  }

  const handleRegister = () => {
    setErrorMessage("")
    registerMutation.mutate({
      username: username.trim(),
      password,
      nickname: nickname.trim() || undefined,
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
    (publicConfigQuery.data?.registration_enabled ?? true) &&
    (publicConfigQuery.data?.password_register_enabled ?? true)

  const passwordLoginEnabled = publicConfigQuery.data?.password_login_enabled ?? true

  const authSources = authSourcesQuery.data ?? []

  // Login button disabled when:
  //   - password login is off, OR
  //   - login mutation is pending, OR
  //   - cap is enabled AND auto-solve mode AND not yet solved (and not in error state)
  // When autoStart=false (manual mode), idle means the user hasn't clicked yet — don't block.
  const loginDisabled =
    !passwordLoginEnabled ||
    loginMutation.isPending ||
    (capEnabled && capAutoSolve && !capReady && !capError)

  return (
    <Card className="w-full border-border/60 bg-background/80 shadow-2xl backdrop-blur">
      <CardContent className="space-y-5 p-5 sm:p-6">
        <div className="space-y-2 text-center">
          <h2 className="text-xl font-semibold tracking-tight text-foreground">
            账号登录
          </h2>
        </div>

        <Tabs value={mode} onValueChange={(value) => setMode(value as "login" | "register")}>
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="login">登录</TabsTrigger>
            <TabsTrigger value="register" disabled={!registrationEnabled}>
              注册
            </TabsTrigger>
          </TabsList>

          <TabsContent value="login" className="space-y-4 pt-4">
            <div className="space-y-3">
              <div className="space-y-2">
                <Input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="用户名"
                  autoComplete="username"
                  onKeyDown={(e) => e.key === "Enter" && handlePasswordLogin()}
                />
                <Input
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  type="password"
                  placeholder="密码"
                  autoComplete="current-password"
                  onKeyDown={(e) => e.key === "Enter" && handlePasswordLogin()}
                />
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
          </TabsContent>

          <TabsContent value="register" className="space-y-4 pt-4">
            <div className="space-y-3">
              <div className="space-y-2">
                <Input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="用户名"
                  autoComplete="username"
                />
                <Input
                  value={nickname}
                  onChange={(e) => setNickname(e.target.value)}
                  placeholder="昵称（可选）"
                  autoComplete="nickname"
                />
                <Input
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  type="password"
                  placeholder="密码（至少 8 位）"
                  autoComplete="new-password"
                />
              </div>

              {errorMessage ? (
                <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                  {errorMessage}
                </div>
              ) : null}

              <Button
                type="button"
                className="w-full"
                variant="secondary"
                onClick={handleRegister}
                disabled={!registrationEnabled || registerMutation.isPending}
              >
                {registerMutation.isPending ? (
                  <>
                    <Spinner className="mr-2" />
                    注册中...
                  </>
                ) : (
                  <>
                    <UserPlus className="mr-2 size-4" />
                    创建账号
                  </>
                )}
              </Button>
            </div>
          </TabsContent>
        </Tabs>

        <Separator />

        {authSources.length > 0 ? (
          authSources.map((source) => (
            <div className="space-y-3" key={source.id}>
              <div className="flex items-center gap-2 text-sm font-medium text-foreground">
                <ShieldCheck className="size-4" />
                第三方认证源
              </div>
              <div className="grid gap-2">
            <Button
              key={source.id}
              type="button"
              variant="outline"
              className="justify-start"
              onClick={() => void handleOAuthLogin(source.name)}
            >
              {source.display_name || source.name} 登录
            </Button>
              </div>
            </div>
          ))
        ) : null}

      </CardContent>
    </Card>
  )
}
