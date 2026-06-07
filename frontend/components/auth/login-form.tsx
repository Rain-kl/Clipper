"use client"

import {useMemo, useState} from "react"
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
import services from "@/lib/services"

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

  const publicConfigQuery = useQuery({
    queryKey: ["public-config"],
    queryFn: services.config.getPublicConfig,
  })

  const authSourcesQuery = useQuery({
    queryKey: ["auth-sources"],
    queryFn: services.auth.getAuthSources,
    enabled: publicConfigQuery.data?.oidc_login_enabled ?? true,
  })

  const redirectTarget = useMemo(
    () => getRedirectTarget(searchParams),
    [searchParams],
  )

  const loginMutation = useMutation({
    mutationFn: services.auth.login,
    onSuccess: (user) => {
      setUser(user)
      router.replace(redirectTarget)
    },
    onError: (error: Error) => {
      setErrorMessage(error.message || "登录失败，请重试")
    },
  })

  const registerMutation = useMutation({
    mutationFn: services.auth.register,
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
    loginMutation.mutate({
      username: username.trim(),
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

  const registrationEnabled =
    (publicConfigQuery.data?.registration_enabled ?? true) &&
    (publicConfigQuery.data?.password_register_enabled ?? true)

  const passwordLoginEnabled = publicConfigQuery.data?.password_login_enabled ?? true

  const authSources = authSourcesQuery.data ?? []

  return (
    <Card className="w-full border-border/60 bg-background/80 shadow-2xl backdrop-blur">
      <CardContent className="space-y-5 p-5 sm:p-6">
        <div className="space-y-2 text-center">
          <h2 className="text-xl font-semibold tracking-tight text-foreground">
            账号登录
          </h2>
          <p className="text-sm text-muted-foreground">
            使用账号密码或第三方 OIDC 认证源登录
          </p>
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
                />
                <Input
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  type="password"
                  placeholder="密码"
                  autoComplete="current-password"
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
                onClick={handlePasswordLogin}
                disabled={!passwordLoginEnabled || loginMutation.isPending}
              >
                {loginMutation.isPending ? (
                  <>
                    <Spinner className="mr-2" />
                    登录中...
                  </>
                ) : (
                  <>
                    <KeyRound className="mr-2 size-4" />
                    账号密码登录
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

        <div className="space-y-3">
          <div className="flex items-center gap-2 text-sm font-medium text-foreground">
            <ShieldCheck className="size-4" />
            第三方认证源
          </div>
          <div className="grid gap-2">
            {authSources.length > 0 ? (
              authSources.map((source) => (
                <Button
                  key={source.id}
                  type="button"
                  variant="outline"
                  className="justify-start"
                  onClick={() => void handleOAuthLogin(source.name)}
                >
                  {source.display_name || source.name} 登录
                </Button>
              ))
            ) : (
              <div className="rounded-lg border border-dashed border-border/60 px-3 py-4 text-sm text-muted-foreground">
                暂无可用认证源
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
