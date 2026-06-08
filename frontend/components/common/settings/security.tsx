"use client"

import {useEffect, useMemo, useState} from "react"
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query"
import {
  Fingerprint,
  Globe,
  Info,
  Loader2,
  Lock,
  Mail,
  Pencil,
  Plus,
  Server,
  Settings,
  Shield,
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
import {Input} from "@/components/ui/input"
import {Label} from "@/components/ui/label"
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog"
import {useAuth} from "@/components/providers/auth-provider"
import {AuthSourceModal} from "@/components/common/settings/auth-source-modal"
import {AdminService, apiConfig} from "@/lib/services"
import type {AuthSource, SystemConfig} from "@/lib/services/admin"
import {toast} from "sonner"
import {SystemStatusManager} from "@/components/common/admin/status"

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
  {
    key: "email_login_verification_enabled",
    title: "邮箱登录验证",
    description: "开启后，使用账号密码登录时需要通过邮箱接收并验证 6 位验证码。",
    icon: Mail,
  },
  {
    key: "email_register_verification_enabled",
    title: "邮箱注册验证",
    description: "开启后，用户注册账号时需要通过邮箱接收并验证 6 位验证码。",
    icon: Mail,
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



export function SecurityMain() {
  const queryClient = useQueryClient()
  const { user, loading } = useAuth()
  const router = useRouter()
  const [authSourceModalOpen, setAuthSourceModalOpen] = useState(false)
  const [selectedSource, setSelectedSource] = useState<AuthSource | null>(null)

  const [capCount, setCapCount] = useState("")
  const [capDifficulty, setCapDifficulty] = useState("")
  const [capSize, setCapSize] = useState("")
  const [capTTL, setCapTTL] = useState("")
  const [capTokenTTL, setCapTokenTTL] = useState("")
  const [capAutoSolve, setCapAutoSolve] = useState(true)
  const [serverAddress, setServerAddress] = useState("")
  const [smtpHost, setSmtpHost] = useState("")
  const [smtpPort, setSmtpPort] = useState("")
  const [smtpUsername, setSmtpUsername] = useState("")
  const [smtpPassword, setSmtpPassword] = useState("")
  const [smtpTestOpen, setSmtpTestOpen] = useState(false)
  const [smtpTestTo, setSmtpTestTo] = useState("")
  const [smtpTestLog, setSmtpTestLog] = useState("")
  const [smtpTestSuccess, setSmtpTestSuccess] = useState<boolean | null>(null)
  const [smtpTestError, setSmtpTestError] = useState("")

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
    if (systemConfigsQuery.data) {
      const cfgMap = systemConfigMap(systemConfigsQuery.data)
      setCapCount(cfgMap["cap_challenge_count"]?.value || "1")
      setCapDifficulty(cfgMap["cap_challenge_difficulty"]?.value || "4")
      setCapSize(cfgMap["cap_challenge_size"]?.value || "32")
      setCapTTL(cfgMap["cap_challenge_ttl_seconds"]?.value || "600")
      setCapTokenTTL(cfgMap["cap_token_ttl_seconds"]?.value || "1200")
      setCapAutoSolve(cfgMap["cap_auto_solve"]?.value !== "false")
      setServerAddress(cfgMap["server_address"]?.value || "")
      setSmtpHost(cfgMap["smtp_host"]?.value || "")
      setSmtpPort(cfgMap["smtp_port"]?.value || "587")
      setSmtpUsername(cfgMap["smtp_username"]?.value || "")
      setSmtpPassword(cfgMap["smtp_password"]?.value || "")
    }
  }, [systemConfigsQuery.data])

  const updateConfigMutation = useMutation({
    mutationFn: async ({ key, value }: { key: string; value: boolean }) => {
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

  const handleToggle = (key: string, checked: boolean) => {
    updateConfigMutation.mutate({ key, value: checked })
  }

  const saveCapMutation = useMutation({
    mutationFn: async () => {
      const updates = [
        { key: "cap_challenge_count", value: capCount },
        { key: "cap_challenge_difficulty", value: capDifficulty },
        { key: "cap_challenge_size", value: capSize },
        { key: "cap_challenge_ttl_seconds", value: capTTL },
        { key: "cap_token_ttl_seconds", value: capTokenTTL },
        { key: "cap_auto_solve", value: capAutoSolve ? "true" : "false" },
      ]

      for (const update of updates) {
        const currentCfg = configs[update.key]
        await AdminService.updateSystemConfig(update.key, {
          value: update.value,
          description: currentCfg?.description || "",
        })
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "system-configs"] })
      toast.success("人机验证配置已成功保存")
    },
    onError: (error: Error) => {
      toast.error(error.message || "保存配置失败")
    },
  })

  const handleCapSave = (e: React.FormEvent) => {
    e.preventDefault()
    saveCapMutation.mutate()
  }

  const saveSystemMutation = useMutation({
    mutationFn: async () => {
      const currentCfg = configs["server_address"]
      await AdminService.updateSystemConfig("server_address", {
        value: serverAddress,
        description: currentCfg?.description || "服务器地址",
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "system-configs"] })
      toast.success("通用配置已成功保存")
    },
    onError: (error: Error) => {
      toast.error(error.message || "保存配置失败")
    },
  })

  const handleSystemSave = (e: React.FormEvent) => {
    e.preventDefault()
    saveSystemMutation.mutate()
  }

  const saveSmtpMutation = useMutation({
    mutationFn: async () => {
      const updates = [
        { key: "smtp_host", value: smtpHost },
        { key: "smtp_port", value: smtpPort },
        { key: "smtp_username", value: smtpUsername },
        { key: "smtp_password", value: smtpPassword },
      ]

      for (const update of updates) {
        const currentCfg = configs[update.key]
        if (update.key === "smtp_password" && (update.value === "" || update.value === "******")) {
          // If already configured and sent empty or mask, skip updating it (keep existing)
          if (currentCfg && currentCfg.value === "******") {
            continue
          }
        }
        await AdminService.updateSystemConfig(update.key, {
          value: update.value,
          description: currentCfg?.description || "",
        })
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "system-configs"] })
      toast.success("SMTP 邮件配置已成功保存")
    },
    onError: (error: Error) => {
      toast.error(error.message || "保存配置失败")
    },
  })

  const handleSmtpSave = (e: React.FormEvent) => {
    e.preventDefault()
    saveSmtpMutation.mutate()
  }

  const testSmtpMutation = useMutation({
    mutationFn: async () => {
      setSmtpTestLog("正在发起连接测试...\n")
      setSmtpTestSuccess(null)
      setSmtpTestError("")

      const res = await AdminService.testSMTP({
        smtp_host: smtpHost,
        smtp_port: parseInt(smtpPort, 10) || 587,
        smtp_username: smtpUsername,
        smtp_password: smtpPassword,
        to: smtpTestTo,
      })
      return res
    },
    onSuccess: (data) => {
      setSmtpTestLog(data.log)
      if (data.success) {
        setSmtpTestSuccess(true)
        toast.success("测试邮件发送成功")
      } else {
        setSmtpTestSuccess(false)
        setSmtpTestError(data.error || "发送失败，请检查配置和日志。")
        toast.error("测试邮件发送失败")
      }
    },
    onError: (error: Error) => {
      setSmtpTestSuccess(false)
      setSmtpTestError(error.message || "请求发送失败")
      setSmtpTestLog((prev) => prev + `\n[请求错误] ${error.message}\n`)
      toast.error(error.message || "测试请求发送失败")
    },
  })

  const handleSmtpTestSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!smtpTestTo) {
      toast.error("请输入目标邮箱地址")
      return
    }
    testSmtpMutation.mutate()
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
            安全设置
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
          <TabsTrigger value="status" className="px-0 pb-2 text-xs font-semibold">
            系统状态
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

            {/* 人机验证配置 (Cap CAPTCHA) */}
            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4 flex flex-row items-center justify-between gap-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
                    <Shield className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">人机验证配置 (Cap CAPTCHA)</CardTitle>
                    <CardDescription className="text-xs">配置基于 Proof-of-Work (PoW) 的无感人机验证，保护系统登录免受暴力破解和撞库攻击</CardDescription>
                  </div>
                </div>
                <Switch
                  checked={configs["cap_login_enabled"]?.value === "true"}
                  disabled={updateConfigMutation.isPending}
                  onCheckedChange={(checked) => handleToggle("cap_login_enabled", checked)}
                />
              </CardHeader>
              <CardContent className="pt-6">
                {/* 自动开始计算 Switch */}
                <div className="flex items-center justify-between rounded-xl border border-dashed p-4 bg-card mb-4">
                  <div className="space-y-0.5">
                    <p className="text-sm font-semibold">打开页面后自动开始计算</p>
                  </div>
                  <Switch
                    checked={capAutoSolve}
                    onCheckedChange={setCapAutoSolve}
                  />
                </div>
                <form onSubmit={handleCapSave} className="space-y-6">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="space-y-1.5">
                      <Label htmlFor="cap_challenge_count" className="text-xs font-semibold">难题数量 (Count)</Label>
                      <Input
                        id="cap_challenge_count"
                        type="number"
                        min={1}
                        max={100}
                        value={capCount}
                        onChange={(e) => setCapCount(e.target.value)}
                        placeholder="50"
                        className="bg-card border-dashed text-xs"
                      />
                      <p className="text-[10px] text-muted-foreground leading-normal">客户端需求解的难题总数。默认 1，推荐 1 至 5</p>
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="cap_challenge_difficulty" className="text-xs font-semibold">验证难度 (Difficulty)</Label>
                      <Input
                        id="cap_challenge_difficulty"
                        type="number"
                        min={1}
                        max={10}
                        value={capDifficulty}
                        onChange={(e) => setCapDifficulty(e.target.value)}
                        placeholder="4"
                        className="bg-card border-dashed text-xs"
                      />
                      <p className="text-[10px] text-muted-foreground leading-normal">PoW 前缀哈希位数，每加 1 计算时间翻倍。默认 4，推荐 4</p>
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="cap_challenge_size" className="text-xs font-semibold">盐值长度 (Size)</Label>
                      <Input
                        id="cap_challenge_size"
                        type="number"
                        min={8}
                        max={64}
                        value={capSize}
                        onChange={(e) => setCapSize(e.target.value)}
                        placeholder="32"
                        className="bg-card border-dashed text-xs"
                      />
                      <p className="text-[10px] text-muted-foreground leading-normal">难题盐值混淆字符长度。默认 32</p>
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="cap_challenge_ttl" className="text-xs font-semibold">难题超时时长 (秒)</Label>
                      <Input
                        id="cap_challenge_ttl"
                        type="number"
                        min={10}
                        value={capTTL}
                        onChange={(e) => setCapTTL(e.target.value)}
                        placeholder="600"
                        className="bg-card border-dashed text-xs"
                      />
                      <p className="text-[10px] text-muted-foreground leading-normal">难题有效期限。默认 600 秒 (10 分钟)</p>
                    </div>

                    <div className="space-y-1.5 sm:col-span-2">
                      <Label htmlFor="cap_token_ttl" className="text-xs font-semibold">验证凭证有效时长 (秒)</Label>
                      <Input
                        id="cap_token_ttl"
                        type="number"
                        min={10}
                        value={capTokenTTL}
                        onChange={(e) => setCapTokenTTL(e.target.value)}
                        placeholder="1200"
                        className="bg-card border-dashed text-xs"
                      />
                      <p className="text-[10px] text-muted-foreground leading-normal">PoW 计算求解通过后，签发的登录凭证有效时长。默认 1200 秒 (20 分钟)</p>
                    </div>
                  </div>

                  <div className="flex justify-end pt-4 border-t border-dashed">
                    <Button
                      type="submit"
                      size="sm"
                      disabled={saveCapMutation.isPending}
                    >
                      {saveCapMutation.isPending ? (
                        <>
                          <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                          保存中...
                        </>
                      ) : (
                        "保存配置"
                      )}
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
        <TabsContent value="operation" />
        <TabsContent value="system" className="pt-4">
          <div className="space-y-6">
            {/* 通用设置 */}
            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
                    <Server className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">通用设置</CardTitle>
                    <CardDescription className="text-xs">配置系统的全局通用参数</CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-6">
                <form onSubmit={handleSystemSave} className="space-y-6">
                  <div className="space-y-1.5">
                    <Label htmlFor="server_address" className="text-xs font-semibold">服务器地址</Label>
                    <Input
                      id="server_address"
                      type="text"
                      value={serverAddress}
                      onChange={(e) => setServerAddress(e.target.value)}
                      placeholder="例如: https://example.com"
                      className="bg-card border-dashed text-xs"
                    />
                    <p className="text-[10px] text-muted-foreground leading-normal">
                      这里可以编辑更改服务器地址。默认不设定，允许从任意源（*）访问 API，此时存在跨域安全风险；如果手动设置服务器地址，CORS 允许源将更新为该地址，消除跨域安全隐患。
                    </p>
                  </div>
                  <div className="flex justify-end pt-4 border-t border-dashed">
                    <Button
                      type="submit"
                      size="sm"
                      disabled={saveSystemMutation.isPending}
                    >
                      {saveSystemMutation.isPending ? (
                        <>
                          <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                          保存中...
                        </>
                      ) : (
                        "保存配置"
                      )}
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>

            {/* SMTP 邮件设置 */}
            <Card className="border border-dashed shadow-sm">
              <CardHeader className="border-b border-dashed pb-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
                    <Mail className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-base font-semibold">SMTP 邮件设置</CardTitle>
                    <CardDescription className="text-xs">配置系统的邮件发送服务 (SMTP)</CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-6">
                <form onSubmit={handleSmtpSave} className="space-y-6">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="space-y-1.5">
                      <Label htmlFor="smtp_host" className="text-xs font-semibold">SMTP 服务器地址</Label>
                      <Input
                        id="smtp_host"
                        type="text"
                        value={smtpHost}
                        onChange={(e) => setSmtpHost(e.target.value)}
                        placeholder="例如: smtp.example.com"
                        className="bg-card border-dashed text-xs"
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="smtp_port" className="text-xs font-semibold">SMTP 端口</Label>
                      <Input
                        id="smtp_port"
                        type="number"
                        value={smtpPort}
                        onChange={(e) => setSmtpPort(e.target.value)}
                        placeholder="例如: 587 或 465"
                        className="bg-card border-dashed text-xs"
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="smtp_username" className="text-xs font-semibold">SMTP 账户</Label>
                      <Input
                        id="smtp_username"
                        type="text"
                        value={smtpUsername}
                        onChange={(e) => setSmtpUsername(e.target.value)}
                        placeholder="例如: sender@example.com"
                        className="bg-card border-dashed text-xs"
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="smtp_password" className="text-xs font-semibold">SMTP 访问凭证</Label>
                      <Input
                        id="smtp_password"
                        type="password"
                        value={smtpPassword}
                        onChange={(e) => setSmtpPassword(e.target.value)}
                        placeholder={configs["smtp_password"]?.value === "******" ? "•••••• (已配置，留空或输入新值)" : "输入凭证密码"}
                        className="bg-card border-dashed text-xs"
                      />
                    </div>
                  </div>

                  <div className="flex justify-end gap-2 pt-4 border-t border-dashed">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setSmtpTestOpen(true)
                        setSmtpTestTo("")
                        setSmtpTestLog("")
                        setSmtpTestSuccess(null)
                        setSmtpTestError("")
                      }}
                      disabled={saveSmtpMutation.isPending}
                    >
                      测试发件
                    </Button>
                    <Button
                      type="submit"
                      size="sm"
                      disabled={saveSmtpMutation.isPending}
                    >
                      {saveSmtpMutation.isPending ? (
                        <>
                          <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                          保存中...
                        </>
                      ) : (
                        "保存配置"
                      )}
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
        <TabsContent value="status" className="pt-4">
          <SystemStatusManager />
        </TabsContent>
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

      <Dialog open={smtpTestOpen} onOpenChange={setSmtpTestOpen}>
        <DialogContent className="max-w-lg border border-dashed">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold">SMTP 发件测试</DialogTitle>
            <DialogDescription className="text-xs">
              输入接收测试邮件的邮箱地址。系统将使用您在表单中当前填写的 SMTP 配置进行发件测试。
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleSmtpTestSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="smtp_test_to" className="text-xs font-semibold">目标邮箱地址</Label>
              <Input
                id="smtp_test_to"
                type="email"
                required
                value={smtpTestTo}
                onChange={(e) => setSmtpTestTo(e.target.value)}
                placeholder="例如: receiver@example.com"
                className="bg-card border-dashed text-xs"
                disabled={testSmtpMutation.isPending}
              />
            </div>

            {smtpTestLog && (
              <div className="space-y-1.5">
                <Label className="text-xs font-semibold">连接与传输日志</Label>
                <pre className="bg-zinc-950 text-zinc-50 font-mono p-4 rounded-lg text-[10px] h-60 overflow-y-auto whitespace-pre-wrap border border-dashed border-zinc-800 leading-relaxed">
                  {smtpTestLog}
                </pre>
              </div>
            )}

            {smtpTestSuccess === true && (
              <div className="p-3 rounded-lg border border-dashed border-emerald-500/30 bg-emerald-500/5 text-emerald-500 text-xs">
                测试成功！邮件已顺利发出。
              </div>
            )}

            {smtpTestSuccess === false && (
              <div className="p-3 rounded-lg border border-dashed border-rose-500/30 bg-rose-500/5 text-rose-500 text-xs break-all">
                测试失败：{smtpTestError}
              </div>
            )}

            <DialogFooter className="gap-2 sm:gap-0 border-t border-dashed pt-4">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setSmtpTestOpen(false)}
                disabled={testSmtpMutation.isPending}
              >
                关闭
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={testSmtpMutation.isPending}
              >
                {testSmtpMutation.isPending ? (
                  <>
                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                    测试中...
                  </>
                ) : (
                  "开始测试"
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </motion.div>
  )
}
