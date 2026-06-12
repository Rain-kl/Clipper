"use client"

import {useEffect, useMemo, useState} from "react"
import {useMutation, useQuery} from "@tanstack/react-query"
import {useRouter, useSearchParams} from "next/navigation"
import {UserPlus} from "lucide-react"
import {toast} from "sonner"
import Link from "next/link"

import {useAuth} from "@/components/providers/auth-provider"
import {Button} from "@/components/ui/button"
import {Input} from "@/components/ui/input"
import {Label} from "@/components/ui/label"
import {Spinner} from "@/components/ui/spinner"
import {Card, CardContent} from "@/components/ui/card"
import services from "@/lib/services"
import type {RegisterRequest} from "@/lib/services/auth/types"

function getRedirectTarget(searchParams: ReturnType<typeof useSearchParams>) {
  const callbackUrl = searchParams.get("callbackUrl")
  const storedRedirect = sessionStorage.getItem("redirect_after_login")
  return callbackUrl || storedRedirect || "/home"
}

function configBool(value: string | undefined, fallback: boolean) {
  if (value === undefined) return fallback
  return value === "true"
}

export function RegisterForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { setUser } = useAuth()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [nickname, setNickname] = useState("")
  const [email, setEmail] = useState("")
  const [code, setCode] = useState("")
  const [registerCooldown, setRegisterCooldown] = useState(0)
  const [errorMessage, setErrorMessage] = useState("")

  useEffect(() => {
    if (registerCooldown > 0) {
      const timer = setTimeout(() => setRegisterCooldown(registerCooldown - 1), 1000)
      return () => clearTimeout(timer)
    }
  }, [registerCooldown])

  const publicConfigQuery = useQuery({
    queryKey: ["public-config"],
    queryFn: () => services.config.getPublicConfig(),
  })

  const redirectTarget = useMemo(
    () => getRedirectTarget(searchParams),
    [searchParams],
  )

  const registrationEnabled =
    configBool(publicConfigQuery.data?.registration_enabled, true) &&
    configBool(publicConfigQuery.data?.password_register_enabled, true)

  const emailRegisterEnabled = configBool(publicConfigQuery.data?.email_register_verification_enabled, false)

  // Redirect to login if registration is closed
  useEffect(() => {
    if (publicConfigQuery.isSuccess && !registrationEnabled) {
      toast.error("系统注册功能已关闭")
      router.replace("/login")
    }
  }, [publicConfigQuery.isSuccess, registrationEnabled, router])

  const registerMutation = useMutation({
    mutationFn: (req: RegisterRequest) => services.auth.register(req),
    onSuccess: (user) => {
      setUser(user)
      router.replace(redirectTarget)
      toast.success("注册并登录成功")
    },
    onError: (error: Error) => {
      setErrorMessage(error.message || "注册失败，请重试")
    },
  })

  const sendRegisterCodeMutation = useMutation({
    mutationFn: (targetEmail: string) => services.auth.sendEmailCode(targetEmail, "register"),
    onSuccess: () => {
      setRegisterCooldown(60)
      toast.success("验证码已发送至您的邮箱，请查收")
    },
    onError: (error: Error) => {
      toast.error(error.message || "发送验证码失败，请重试")
    },
  })

  const handleSendRegisterCode = () => {
    const trimmedEmail = email.trim()
    if (!trimmedEmail) {
      toast.error("请先输入邮箱地址")
      return
    }
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRegex.test(trimmedEmail)) {
      toast.error("请输入有效的邮箱地址")
      return
    }
    sendRegisterCodeMutation.mutate(trimmedEmail)
  }

  const handleRegister = () => {
    setErrorMessage("")
    if (!username.trim() || !password) {
      toast.error("用户名和密码不能为空")
      return
    }
    if (password.length < 8) {
      toast.error("密码长度不能少于 8 位")
      return
    }
    const trimmedEmail = email.trim()
    if (!trimmedEmail) {
      toast.error("邮箱不能为空")
      return
    }
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRegex.test(trimmedEmail)) {
      toast.error("请输入有效的邮箱地址")
      return
    }
    if (emailRegisterEnabled && !code.trim()) {
      toast.error("验证码不能为空")
      return
    }
    registerMutation.mutate({
      username: username.trim(),
      password,
      nickname: nickname.trim() || undefined,
      email: trimmedEmail,
      code: code.trim() || undefined,
    })
  }

  if (publicConfigQuery.isPending) {
    return (
      <Card className="w-full border-border/60 bg-background/80 shadow-2xl backdrop-blur">
        <CardContent className="flex justify-center items-center py-12">
          <Spinner />
        </CardContent>
      </Card>
    )
  }

  if (!registrationEnabled) {
    return null
  }

  return (
    <Card className="w-full border-border/60 bg-background/80 shadow-2xl backdrop-blur">
      <CardContent className="space-y-5 p-5 sm:p-6">
        <div className="space-y-2 text-center">
          <h2 className="text-xl font-semibold tracking-tight text-foreground">
            创建账号
          </h2>
        </div>

        <div className="space-y-3 pt-2">
          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入用户名"
                autoComplete="username"
                onKeyDown={(e) => e.key === "Enter" && handleRegister()}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="nickname">
                昵称
                <span className="text-muted-foreground font-normal text-xs ml-1">（可选）</span>
              </Label>
              <Input
                id="nickname"
                value={nickname}
                onChange={(e) => setNickname(e.target.value)}
                placeholder="请输入昵称"
                autoComplete="nickname"
                onKeyDown={(e) => e.key === "Enter" && handleRegister()}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="password">密码</Label>
              <Input
                id="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                type="password"
                placeholder="请输入密码（至少 8 位）"
                autoComplete="new-password"
                onKeyDown={(e) => e.key === "Enter" && handleRegister()}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="email">电子邮箱</Label>
              <Input
                id="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="请输入电子邮箱"
                autoComplete="email"
                onKeyDown={(e) => e.key === "Enter" && handleRegister()}
              />
            </div>
            {emailRegisterEnabled && (
              <div className="space-y-1.5">
                <Label htmlFor="code">邮箱验证码</Label>
                <div className="flex gap-2">
                  <Input
                    id="code"
                    value={code}
                    onChange={(e) => setCode(e.target.value)}
                    placeholder="请输入 6 位邮箱验证码"
                    maxLength={6}
                    className="flex-1"
                    onKeyDown={(e) => e.key === "Enter" && handleRegister()}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleSendRegisterCode}
                    disabled={registerCooldown > 0 || sendRegisterCodeMutation.isPending}
                    className="w-[120px] text-xs"
                  >
                    {registerCooldown > 0 ? `${registerCooldown}秒后重发` : "获取验证码"}
                  </Button>
                </div>
              </div>
            )}
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
            disabled={registerMutation.isPending}
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

        <div className="text-center text-xs text-muted-foreground mt-4">
          Already have an account?{" "}
          <Link href="/login" className="font-semibold text-indigo-500 hover:text-indigo-600 transition-colors">
            Sign in
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}
