"use client"

import {useEffect, useMemo, useState} from "react"
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query"
import {
  CalendarClock,
  Fingerprint,
  Globe,
  Info,
  Loader2,
  Lock,
  Monitor,
  Pencil,
  Plus,
  Server,
  Settings,
  Trash2,
  UserPlus
} from "lucide-react"
import {useRouter} from "next/navigation"
import {motion} from "motion/react"
import packageJson from "../../../package.json"

import {Button} from "@/components/ui/button"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Switch} from "@/components/ui/switch"
import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {useAuth} from "@/components/providers/auth-provider"
import {AuthSourceModal} from "@/components/common/settings/auth-source-modal"
import {AdminService, apiConfig} from "@/lib/services"
import type {AuthSource, SystemConfig} from "@/lib/services/admin"
import {toast} from "sonner"

const SECURITY_KEYS = [
  {
    key: "password_login_enabled",
    title: "允许密码登录",
    description: "关闭后仅保留第三方 OIDC 认证源进行系统登录。",
    icon: Lock,
  },
  {
    key: "registration_enabled",
    title: "允许注册",
    description: "关闭后系统将禁止新用户进行自主账号注册。",
    icon: UserPlus,
  },
  {
    key: "password_register_enabled",
    title: "允许密码注册",
    description: "关闭后只能通过管理员创建或第三方认证关联建号。",
    icon: Fingerprint,
  },
  {
    key: "oidc_login_enabled",
    title: "允许 OIDC 登录",
    description: "关闭后所有的第三方 OIDC 认证登录入口都会被隐藏。",
    icon: Globe,
  },
] as const

type SecurityKey = (typeof SECURITY_KEYS)[number]["key"]

function systemConfigMap(configs: SystemConfig[]) {
  return configs.reduce<Record<string, SystemConfig>>((accumulator, config) => {
    accumulator[config.key] = config
    return accumulator
  }, {})
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-dashed py-2 last:border-b-0">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-right text-xs font-medium text-foreground break-all">{value || "-"}</span>
    </div>
  )
}

function formatBooleanConfig(config?: SystemConfig) {
  if (!config) return "未配置"
  return config.value === "true" ? "启用" : "禁用"
}

export function SecurityMain() {
  const queryClient = useQueryClient()
  const { user, loading } = useAuth()
  const router = useRouter()
  const [authSourceModalOpen, setAuthSourceModalOpen] = useState(false)
  const [selectedSource, setSelectedSource] = useState<AuthSource | null>(null)
  const [runtimeInfo, setRuntimeInfo] = useState({
    language: "-",
    platform: "-",
    timezone: "-",
    viewport: "-",
    userAgent: "-",
  })

  const systemConfigsQuery = useQuery({
    queryKey: ["admin", "system-configs"],
    queryFn: () => AdminService.listSystemConfigs("system"),
    enabled: !!user?.is_admin,
  })

  const authSourcesQuery = useQuery({
    queryKey: ["auth", "sources"],
    queryFn: () => AdminService.listAuthSources(),
    enabled: !!user?.is_admin,
  })

  const configs = useMemo(
    () => systemConfigMap(systemConfigsQuery.data ?? []),
    [systemConfigsQuery.data],
  )

  useEffect(() => {
    if (!loading && (!user || !user.is_admin)) {
      router.replace("/settings/profile")
    }
  }, [user, loading, router])

  useEffect(() => {
    setRuntimeInfo({
      language: navigator.language || "-",
      platform: navigator.platform || "-",
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "-",
      viewport: `${window.innerWidth} x ${window.innerHeight}`,
      userAgent: navigator.userAgent || "-",
    })
  }, [])

  const updateConfigMutation = useMutation({
    mutationFn: async ({ key, value }: { key: SecurityKey; value: boolean }) => {
      const config = configs[key]
      if (!config) {
        throw new Error(`缺少配置项: ${key}`)
      }
      await AdminService.updateSystemConfig(key, {
        value: value ? "true" : "false",
        description: config.description,
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "system-configs"] })
      toast.success("系统安全配置已更新")
    },
    onError: (error: Error) => {
      toast.error(error.message || "更新配置失败")
    },
  })

  const toggleSourceMutation = useMutation({
    mutationFn: async (source: AuthSource) => {
      await AdminService.toggleAuthSource(source.id, { is_active: !source.is_active })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["auth", "sources"] })
      await queryClient.invalidateQueries({ queryKey: ["auth", "public-sources"] })
      toast.success("认证源状态已更新")
    },
    onError: (error: Error) => {
      toast.error(error.message || "切换状态失败")
    },
  })

  const deleteSourceMutation = useMutation({
    mutationFn: async (sourceId: string) => {
      await AdminService.deleteAuthSource(sourceId)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["auth", "sources"] })
      await queryClient.invalidateQueries({ queryKey: ["auth", "public-sources"] })
      toast.success("认证源已删除")
    },
    onError: (error: Error) => {
      toast.error(error.message || "删除认证源失败")
    },
  })

  const handleToggle = (key: SecurityKey, checked: boolean) => {
    updateConfigMutation.mutate({ key, value: checked })
  }

  if (loading || !user || !user.is_admin) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="size-6 animate-spin text-indigo-500" />
      </div>
    )
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: "easeOut" }}
      className="py-6 space-y-6 max-w-4xl mx-auto px-4"
    >
      <Tabs defaultValue="security" className="w-full">
        <TabsList className="w-full overflow-x-auto">
          <TabsTrigger value="security" className="px-0 pb-2 text-xs font-semibold">
            系统安全设置
          </TabsTrigger>
          <TabsTrigger value="operation" className="px-0 pb-2 text-xs font-semibold">
            运营设置
          </TabsTrigger>
          <TabsTrigger value="system" className="px-0 pb-2 text-xs font-semibold">
            系统设置
          </TabsTrigger>
          <TabsTrigger value="other" className="px-0 pb-2 text-xs font-semibold">
            其他设置
          </TabsTrigger>
          <TabsTrigger value="info" className="px-0 pb-2 text-xs font-semibold">
            系统信息
          </TabsTrigger>
        </TabsList>

        <TabsContent value="security" className="pt-4">
          <div className="space-y-6">
            {/* 系统登录与注册控制 */}
            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
                    <Settings className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">系统安全与注册控制</CardTitle>
                    <CardDescription className="text-xs">配置系统的登录限制与用户自主注册权限</CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {SECURITY_KEYS.map((item) => {
                    const config = configs[item.key]
                    const checked = config ? config.value === "true" : false
                    const Icon = item.icon
                    return (
                      <div
                        key={item.key}
                        className="flex items-center justify-between gap-4 rounded-xl border border-dashed p-4 bg-card hover:bg-muted/10 hover:border-indigo-500/30 transition-all duration-300 shadow-sm"
                      >
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            {Icon && <Icon className="size-4 text-indigo-500" />}
                            <span className="font-medium text-sm text-foreground">{item.title}</span>
                          </div>
                          <p className="text-xs text-muted-foreground leading-relaxed pr-2">{item.description}</p>
                        </div>
                        <Switch
                          checked={checked}
                          disabled={updateConfigMutation.isPending}
                          onCheckedChange={(value) => handleToggle(item.key, value)}
                        />
                      </div>
                    )
                  })}
                </div>
              </CardContent>
            </Card>

            {/* 认证源配置管理 */}
            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4 flex flex-row items-center justify-between gap-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
                    <Globe className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">认证源管理</CardTitle>
                    <CardDescription className="text-xs">添加、修改并启用系统自定义的 OIDC 认证源</CardDescription>
                  </div>
                </div>
                <Button
                  type="button"
                  size="sm"
                  onClick={() => {
                    setSelectedSource(null)
                    setAuthSourceModalOpen(true)
                  }}
                  variant="secondary"
                >
                  <Plus className="mr-1.5 size-3.5" />
                  新增认证源
                </Button>
              </CardHeader>
              <CardContent className="pt-6 space-y-3">
                {authSourcesQuery.isPending ? (
                  <div className="flex items-center justify-center p-8">
                    <Loader2 className="size-6 animate-spin text-muted-foreground/50" />
                  </div>
                ) : (authSourcesQuery.data ?? []).length > 0 ? (
                  (authSourcesQuery.data ?? []).map((source) => (
                    <div
                      key={source.id}
                      className="flex items-center justify-between rounded-xl border border-dashed p-4 bg-card hover:bg-muted/10 transition-all duration-300 shadow-sm"
                    >
                      <div className="space-y-1.5">
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-sm text-foreground">{source.display_name || source.name}</span>
                          <span className={`text-[10px] px-2 py-0.5 rounded-full border font-medium ${
                            source.is_active
                              ? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20"
                              : "bg-amber-500/10 text-amber-500 border-amber-500/20"
                          }`}>
                            {source.is_active ? "已启用" : "已禁用"}
                          </span>
                        </div>
                        <div className="text-xs text-muted-foreground font-mono">
                          标识符: {source.name} · 类型: {source.type.toUpperCase()}
                        </div>
                      </div>
                      <div className="flex items-center gap-4">
                        <span className={`text-xs px-2.5 py-1 rounded-lg border font-medium hidden sm:inline-block ${
                          source.client_secret_configured
                            ? "bg-indigo-500/5 text-indigo-500 border-indigo-500/10"
                            : "bg-rose-500/5 text-rose-500 border-rose-500/10"
                        }`}>
                          {source.client_secret_configured ? "Secret 已配置" : "Secret 未配置"}
                        </span>

                        <div className="flex items-center gap-2">
                          <Switch
                            checked={source.is_active}
                            disabled={toggleSourceMutation.isPending}
                            className="scale-90 mr-2"
                            onCheckedChange={() => toggleSourceMutation.mutate(source)}
                          />
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="size-8 text-muted-foreground hover:text-indigo-500 hover:bg-indigo-500/10 rounded-lg transition-colors"
                            onClick={() => {
                              setSelectedSource(source)
                              setAuthSourceModalOpen(true)
                            }}
                          >
                            <Pencil className="size-4" />
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="size-8 text-muted-foreground hover:text-rose-500 hover:bg-rose-500/10 rounded-lg transition-colors"
                            disabled={deleteSourceMutation.isPending}
                            onClick={() => {
                              if (window.confirm(`确定删除认证源「${source.display_name || source.name}」吗？`)) {
                                deleteSourceMutation.mutate(source.id)
                              }
                            }}
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        </div>
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="rounded-xl border border-dashed border-border/50 px-4 py-8 text-center text-xs text-muted-foreground bg-muted/5 flex flex-col items-center justify-center gap-3">
                    <span>暂无配置的认证源，点击上方按钮新增</span>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setSelectedSource(null)
                        setAuthSourceModalOpen(true)
                      }}
                      className="border-dashed"
                    >
                      <Plus className="mr-1.5 size-3.5" />
                      新增认证源
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>
        <TabsContent value="operation" />
        <TabsContent value="system" />
        <TabsContent value="other" />
        <TabsContent value="info" className="pt-4">
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-muted text-muted-foreground">
                    <Info className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">应用信息</CardTitle>
                    <CardDescription className="text-xs">当前前端应用的版本与构建信息</CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-4">
                <InfoRow label="应用名称" value={packageJson.name} />
                <InfoRow label="版本号" value={packageJson.version} />
                <InfoRow label="构建时间" value={packageJson.buildDate} />
                <InfoRow label="Next.js" value={packageJson.dependencies.next} />
                <InfoRow label="React" value={packageJson.dependencies.react} />
              </CardContent>
            </Card>

            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-muted text-muted-foreground">
                    <Server className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">服务连接</CardTitle>
                    <CardDescription className="text-xs">前端 API 客户端的基础连接参数</CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-4">
                <InfoRow label="API Base URL" value={apiConfig.baseURL || "同源"} />
                <InfoRow label="请求超时" value={`${apiConfig.timeout}ms`} />
                <InfoRow label="携带凭证" value={apiConfig.withCredentials ? "是" : "否"} />
                <InfoRow label="系统配置项" value={`${systemConfigsQuery.data?.length ?? 0} 项`} />
                <InfoRow label="认证源数量" value={`${authSourcesQuery.data?.length ?? 0} 个`} />
              </CardContent>
            </Card>

            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-muted text-muted-foreground">
                    <Monitor className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">运行环境</CardTitle>
                    <CardDescription className="text-xs">当前浏览器会话的本地运行信息</CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-4">
                <InfoRow label="语言" value={runtimeInfo.language} />
                <InfoRow label="平台" value={runtimeInfo.platform} />
                <InfoRow label="时区" value={runtimeInfo.timezone} />
                <InfoRow label="视口" value={runtimeInfo.viewport} />
                <InfoRow label="User Agent" value={runtimeInfo.userAgent} />
              </CardContent>
            </Card>

            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-muted text-muted-foreground">
                    <CalendarClock className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">安全配置概览</CardTitle>
                    <CardDescription className="text-xs">当前系统登录与注册开关状态</CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-4">
                <InfoRow label="密码登录" value={formatBooleanConfig(configs.password_login_enabled)} />
                <InfoRow label="开放注册" value={formatBooleanConfig(configs.registration_enabled)} />
                <InfoRow label="密码注册" value={formatBooleanConfig(configs.password_register_enabled)} />
                <InfoRow label="OIDC 登录" value={formatBooleanConfig(configs.oidc_login_enabled)} />
                <InfoRow label="配置加载状态" value={systemConfigsQuery.isFetching ? "刷新中" : "已加载"} />
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>

      <AuthSourceModal
        isOpen={authSourceModalOpen}
        source={selectedSource}
        onClose={() => setAuthSourceModalOpen(false)}
        onChanged={async () => {
          await queryClient.invalidateQueries({ queryKey: ["auth", "sources"] })
          await queryClient.invalidateQueries({ queryKey: ["auth", "public-sources"] })
          await authSourcesQuery.refetch()
        }}
      />
    </motion.div>
  )
}
