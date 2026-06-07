"use client"

import Link from "next/link"
import {Card, CardContent, CardDescription, CardTitle} from "@/components/ui/card"
import {Bell, Key, Loader2, Palette, Shield, UserRound} from "lucide-react"
import {useAuth} from "@/components/providers/auth-provider"

/* 设置项 */
const settingsItems = [
  {
    title: "个人资料",
    description: "查看您的账户信息",
    icon: UserRound,
    href: "/settings/profile",
    category: "个人设置",
  },
  {
    title: "访问令牌 (AccessToken)",
    description: "管理用于 API 访问的个人 Token",
    icon: Key,
    href: "/settings/access-token",
    category: "个人设置",
  },
  {
    title: "通知设置",
    description: "设置您的通知偏好",
    icon: Bell,
    href: "/settings/notifications",
    category: "账户设置",
  },
  {
    title: "安全设置",
    description: "管理您的账户安全",
    icon: Shield,
    href: "/settings/security",
    category: "账户设置",
  },
  {
    title: "外观设置",
    description: "自定义界面主题 and 显示",
    icon: Palette,
    href: "/settings/appearance",
    category: "个人设置",
  },
]

export default function SettingsPage() {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[200px]">
        <Loader2 className="size-6 animate-spin text-indigo-500" />
      </div>
    )
  }

  // 非管理员过滤掉安全设置
  const filteredItems = settingsItems.filter((item) => {
    if (item.href === "/settings/security") {
      return !!user?.is_admin
    }
    return true
  })

  const groupedSettings = filteredItems.reduce((acc, item) => {
    if (!acc[item.category]) {
      acc[item.category] = []
    }
    acc[item.category].push(item)
    return acc
  }, {} as Record<string, typeof settingsItems>)

  return (
    <div className="space-y-6 py-6">
      <div className="font-semibold text-lg">设置</div>

      {Object.entries(groupedSettings).map(([category, items]) => (
        <div key={category} className="space-y-4">
          <div className="font-medium text-sm text-muted-foreground">{category}</div>
          <div className="grid grid-cols-2 gap-4">
            {items.map((item) => (
              <Link key={item.href} href={item.href}>
                <Card className="py-2 border border-dashed shadow-none hover:bg-muted/50 transition-colors cursor-pointer h-full">
                  <CardContent>
                    <div className="flex items-center gap-4">
                      <item.icon className="size-5 text-primary" />
                      <div>
                        <CardTitle className="mb-1 text-sm">{item.title}</CardTitle>
                        <CardDescription className="text-xs">
                          {item.description}
                        </CardDescription>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </Link>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
