"use client"

import * as React from "react"
import Link from "next/link"
import { motion, useAnimation } from "motion/react"
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar"
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb"
import { useUser } from "@/contexts/user-context"
import { Shield } from "lucide-react"

export function ProfileMain() {
  const { user, loading, getTrustLevelLabel } = useUser()
  const controls = useAnimation()
  const isAnimatingRef = React.useRef(false)

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
    </div>
  )
}
