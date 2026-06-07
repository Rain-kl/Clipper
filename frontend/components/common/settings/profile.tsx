"use client"

import * as React from "react"
import Link from "next/link"
import {motion, useAnimation} from "motion/react"
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query"
import {Avatar, AvatarFallback, AvatarImage} from "@/components/ui/avatar"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator
} from "@/components/ui/breadcrumb"
import {useUser} from "@/contexts/user-context"
import {ArrowRight, Link2, Loader2, Shield, Unlink} from "lucide-react"
import {Button} from "@/components/ui/button"
import {Separator} from "@/components/ui/separator"
import {AuthService} from "@/lib/services"
import {toast} from "sonner"

export function ProfileMain() {
  const { user, loading, getTrustLevelLabel } = useUser()
  const controls = useAnimation()
  const isAnimatingRef = React.useRef(false)
  const queryClient = useQueryClient()

  const externalAccountBindingsQuery = useQuery({
    queryKey: ["auth", "external-accounts"],
    queryFn: () => AuthService.getExternalAccountBindings(),
  })

  const publicAuthSourcesQuery = useQuery({
    queryKey: ["auth", "public-sources"],
    queryFn: () => AuthService.getAuthSources(),
  })

  const bindSourceMutation = useMutation({
    mutationFn: async (sourceName: string) => {
      const { authorize_url } = await AuthService.getAuthorizeUrl(sourceName, "bind")
      sessionStorage.setItem("redirect_after_login", `${window.location.pathname}${window.location.search}`)
      window.location.href = authorize_url
    },
    onError: (error: Error) => {
      toast.error(error.message || "绑定认证源失败")
    },
  })

  const handleAvatarClick = () => {
    if (isAnimatingRef.current) return

    isAnimatingRef.current = true
    controls.start({
      rotate: [0, -20, 20, -20, 20, 0],
      transition: { duration: 0.5, ease: "easeInOut" }
    })

    setTimeout(() => {
      isAnimatingRef.current = false
    }, 650)
  }

  if (loading) {
    return (
      <div className="py-6 space-y-4">
        <div className="border-b border-border pb-4">
          <h1 className="text-2xl font-semibold">个人资料</h1>
        </div>
      </div>
    )
  }

  if (!user) {
    return (
      <div className="py-6 space-y-6">
        <div className="text-sm text-muted-foreground">未找到用户信息</div>
      </div>
    )
  }

  return (
    <div className="py-6 space-y-6 max-w-4xl mx-auto">
      <div className="font-semibold">
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link href="/settings" className="text-base text-primary">设置</Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              <BreadcrumbPage className="text-base font-semibold">个人资料</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>

      <div className="space-y-6 bg-card border border-dashed rounded-lg p-6">
        <div className="border-b pb-4">
          <h2 className="text-lg font-semibold tracking-tight">基本资料</h2>
          <p className="text-xs text-muted-foreground">您的个人账户基本信息</p>
        </div>

        <div className="flex flex-col sm:flex-row items-center sm:items-start gap-6 pt-2">
          <motion.div
            animate={controls}
            onClick={handleAvatarClick}
            className="cursor-pointer origin-center shrink-0"
            whileHover={{ scale: 1.05 }}
          >
            <Avatar className="size-20 md:size-24 border-2 border-primary/10 shadow-md">
              <AvatarImage src={user.avatar_url} alt={user.nickname || user.username} />
              <AvatarFallback className="text-2xl bg-indigo-600 text-white font-bold">
                {(user.nickname || user.username).slice(0, 2).toUpperCase()}
              </AvatarFallback>
            </Avatar>
          </motion.div>

          <div className="flex-1 w-full space-y-6">
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">账户</div>
                <div className="text-sm font-semibold">@{user.username}</div>
              </div>

              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">昵称</div>
                <div className="text-sm font-semibold">{user.nickname || '未设置'}</div>
              </div>

              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">用户ID (UID)</div>
                <div className="text-sm font-mono font-semibold">{user.id}</div>
              </div>

              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">信任等级</div>
                <div className="text-sm font-semibold">{getTrustLevelLabel(user.trust_level)}</div>
              </div>

              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">管理员身份</div>
                <div className="text-sm font-semibold flex items-center gap-1">
                  {user.is_admin ? (
                    <span className="text-rose-600 flex items-center gap-1">
                      <Shield className="size-3.5" />
                      是
                    </span>
                  ) : (
                    <span>否</span>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 账号绑定面板 */}
      <div className="space-y-6 bg-card border border-dashed rounded-lg p-6">
        <div className="border-b pb-4 flex items-center gap-2">
          <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
            <Link2 className="size-4" />
          </div>
          <div>
            <h2 className="text-lg font-semibold tracking-tight">第三方账号绑定</h2>
            <p className="text-xs text-muted-foreground">管理并关联您的第三方授权账户，便于快捷登录与验证</p>
          </div>
        </div>

        {/* 已绑定账号列表 */}
        <div className="space-y-3 pt-2">
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">已绑定账号</h3>
          {externalAccountBindingsQuery.isPending ? (
            <div className="flex items-center justify-center py-6">
              <Loader2 className="size-5 animate-spin text-indigo-500" />
            </div>
          ) : (externalAccountBindingsQuery.data ?? []).length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {(externalAccountBindingsQuery.data ?? []).map((binding) => (
                <div
                  key={binding.id}
                  className="flex items-center justify-between gap-4 rounded-xl border border-dashed p-4 bg-card hover:bg-muted/10 transition-all duration-300"
                >
                  <div className="space-y-1">
                    <span className="font-semibold text-xs text-foreground block">{binding.auth_source_label}</span>
                    <span className="text-xs text-muted-foreground font-mono block truncate max-w-[180px]">
                      {binding.external_username || binding.email || "未提供账号标识"}
                    </span>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="text-xs text-muted-foreground hover:text-rose-500 hover:bg-rose-500/10 rounded-lg h-8 px-2.5 transition-colors"
                    onClick={async () => {
                      await AuthService.deleteExternalAccountBinding(binding.id)
                      await queryClient.invalidateQueries({ queryKey: ["auth", "external-accounts"] })
                      toast.success("绑定已移除")
                    }}
                  >
                    <Unlink className="size-3.5 mr-1" />
                    解除绑定
                  </Button>
                </div>
              ))}
            </div>
          ) : (
            <div className="rounded-xl border border-dashed border-border/50 px-4 py-8 text-center text-xs text-muted-foreground bg-muted/5 flex flex-col items-center justify-center">
              <Link2 className="size-6 text-muted-foreground/30 mb-2" />
              暂无绑定的第三方账号
            </div>
          )}
        </div>

        <Separator className="border-dashed" />

        {/* 绑定新账号列表 */}
        <div className="space-y-3">
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">绑定新账号</h3>
          {publicAuthSourcesQuery.isPending ? (
            <div className="flex items-center justify-center py-6">
              <Loader2 className="size-5 animate-spin text-indigo-500" />
            </div>
          ) : (publicAuthSourcesQuery.data ?? []).length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {(publicAuthSourcesQuery.data ?? []).map((source) => (
                <Button
                  key={source.id}
                  type="button"
                  variant="outline"
                  className="flex items-center justify-between w-full border border-dashed rounded-xl px-4 py-3.5 text-left font-normal text-xs hover:bg-indigo-500/5 hover:text-indigo-500 hover:border-indigo-500/30 transition-all duration-300 group h-auto"
                  onClick={() => {
                    void bindSourceMutation.mutateAsync(source.name)
                  }}
                >
                  <div className="flex items-center gap-2">
                    <Link2 className="size-3.5 text-muted-foreground group-hover:text-indigo-500" />
                    <span>绑定 {source.display_name || source.name}</span>
                  </div>
                  <ArrowRight className="size-3.5 opacity-0 -translate-x-1 group-hover:opacity-100 group-hover:translate-x-0 transition-all text-indigo-500" />
                </Button>
              ))}
            </div>
          ) : (
            <div className="text-xs text-muted-foreground text-center py-4">
              暂无可用第三方认证源
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
