"use client"

import * as React from "react"
import Link from "next/link"
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query"
import {AnimatePresence, motion} from "motion/react"
import {
  Download,
  FileArchive,
  FileAudio,
  FileImage,
  FileText,
  FileVideo,
  Folder,
  Loader2,
  Search,
  Trash2,
  Upload,
  X,
} from "lucide-react"
import {toast} from "sonner"

import {Button} from "@/components/ui/button"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import {Input} from "@/components/ui/input"
import {Badge} from "@/components/ui/badge"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {formatFileSize, UploadService} from "@/lib/services/upload/upload.service"
import type {Upload as UploadRecord} from "@/lib/services/upload/types"

/* ─── 工具函数 ─────────────────────────────────────────── */

function getFileIcon(mimeType: string, className = "size-8") {
  if (mimeType.startsWith("image/")) return <FileImage className={`${className} text-blue-400`} />
  if (mimeType.startsWith("video/")) return <FileVideo className={`${className} text-purple-400`} />
  if (mimeType.startsWith("audio/")) return <FileAudio className={`${className} text-green-400`} />
  if (mimeType.includes("zip") || mimeType.includes("tar") || mimeType.includes("gzip"))
    return <FileArchive className={`${className} text-amber-400`} />
  return <FileText className={`${className} text-slate-400`} />
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  })
}

/* ─── 文件管理主组件 ────────────────────────────────────── */

export function FilesMain() {
  const queryClient = useQueryClient()
  const [keyword, setKeyword] = React.useState("")
  const [debouncedKeyword, setDebouncedKeyword] = React.useState("")
  const [selectedIds, setSelectedIds] = React.useState<Set<string>>(new Set())
  const [deleteTarget, setDeleteTarget] = React.useState<UploadRecord | null>(null)
  const [page, setPage] = React.useState(1)
  const pageSize = 24

  // 搜索防抖
  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedKeyword(keyword)
      setPage(1)
    }, 400)
    return () => clearTimeout(timer)
  }, [keyword])

  // 文件列表查询
  const listQuery = useQuery({
    queryKey: ["files", "my", page, pageSize, debouncedKeyword],
    queryFn: () => UploadService.listMyFiles(page, pageSize, debouncedKeyword || undefined),
  })

  const files = listQuery.data?.items ?? []
  const total = listQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / pageSize)

  // 删除单文件
  const deleteMutation = useMutation({
    mutationFn: (id: string) => UploadService.deleteFile(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["files", "my"] })
      toast.success("文件已删除")
      setDeleteTarget(null)
    },
    onError: (err: Error) => toast.error(err.message || "删除失败"),
  })

  // 批量 ZIP 下载
  const batchDownloadMutation = useMutation({
    mutationFn: (ids: string[]) => UploadService.batchDownload(ids),
    onSuccess: (blob) => {
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = "batch_download.zip"
      a.click()
      URL.revokeObjectURL(url)
      toast.success("批量下载已开始")
    },
    onError: () => toast.error("批量下载失败"),
  })

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  const clearSelection = () => setSelectedIds(new Set())

  const selectAll = () => setSelectedIds(new Set(files.map((f) => f.id)))

  const handleDownload = (file: UploadRecord) => {
    const url = UploadService.getDownloadUrl(file.id)
    const a = document.createElement("a")
    a.href = url
    a.download = file.file_name
    a.click()
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: "easeOut" }}
      className="py-6 space-y-6 max-w-6xl mx-auto"
    >
      {/* Breadcrumb */}
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
              <BreadcrumbPage className="text-base font-semibold">文件管理</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>

      {/* 头部 */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b pb-5">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-sky-500 to-indigo-600 text-white shadow-lg shadow-sky-500/20">
            <Folder className="size-6" />
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight bg-gradient-to-r from-foreground via-foreground/90 to-muted-foreground bg-clip-text text-transparent">
              我的文件
            </h1>
            <p className="text-sm text-muted-foreground">
              管理您上传的所有文件，支持下载和批量操作
            </p>
          </div>
        </div>

        {/* 操作按钮区 */}
        <div className="flex items-center gap-2 shrink-0">
          <AnimatePresence>
            {selectedIds.size > 0 && (
              <motion.div
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.9 }}
                className="flex items-center gap-2"
              >
                <Badge variant="secondary" className="text-xs px-2.5">
                  已选 {selectedIds.size} 个
                </Badge>
                <Button
                  size="sm"
                  variant="outline"
                  className="border-dashed text-xs h-8"
                  onClick={() => batchDownloadMutation.mutate([...selectedIds])}
                  disabled={batchDownloadMutation.isPending}
                >
                  {batchDownloadMutation.isPending ? (
                    <Loader2 className="size-3.5 mr-1 animate-spin" />
                  ) : (
                    <FileArchive className="size-3.5 mr-1" />
                  )}
                  打包下载
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-xs h-8 px-2"
                  onClick={clearSelection}
                >
                  <X className="size-3.5" />
                </Button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>

      {/* 搜索栏 */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
          <Input
            placeholder="搜索文件名..."
            className="pl-8 h-8 text-xs border-dashed rounded-lg focus-visible:ring-0"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
          {keyword && (
            <button
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              onClick={() => setKeyword("")}
            >
              <X className="size-3" />
            </button>
          )}
        </div>
        {files.length > 0 && (
          <Button
            size="sm"
            variant="ghost"
            className="text-xs h-8 border border-dashed text-muted-foreground"
            onClick={selectedIds.size === files.length ? clearSelection : selectAll}
          >
            {selectedIds.size === files.length ? "取消全选" : "全选本页"}
          </Button>
        )}
        {total > 0 && (
          <span className="text-xs text-muted-foreground shrink-0">共 {total} 个文件</span>
        )}
      </div>

      {/* 文件网格 */}
      {listQuery.isPending ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="size-6 animate-spin text-sky-500" />
        </div>
      ) : files.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 gap-4 text-muted-foreground">
          <Upload className="size-12 text-muted-foreground/30" />
          <p className="text-sm">
            {debouncedKeyword ? "没有匹配的文件" : "您还没有上传任何文件"}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
          {files.map((file, idx) => {
            const isSelected = selectedIds.has(file.id)
            return (
              <motion.div
                key={file.id}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.25, delay: idx * 0.02 }}
                className={`group relative rounded-xl border border-dashed p-3 flex flex-col gap-2 cursor-pointer transition-all duration-200 hover:bg-muted/40 hover:shadow-sm select-none ${
                  isSelected
                    ? "bg-sky-500/8 border-sky-500/40 ring-1 ring-sky-500/30"
                    : "bg-card hover:border-muted-foreground/25"
                }`}
                onClick={() => toggleSelect(file.id)}
              >
                {/* 选中指示 */}
                {isSelected && (
                  <div className="absolute top-2 left-2 size-4 rounded-full bg-sky-500 flex items-center justify-center z-10">
                    <svg className="size-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                )}

                {/* 文件图标 / 图片预览 */}
                <div className="flex items-center justify-center h-14 rounded-lg bg-muted/30 overflow-hidden">
                  {file.mime_type.startsWith("image/") ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={`/f/${file.id}`}
                      alt={file.file_name}
                      className="h-full w-full object-cover rounded-lg"
                      onError={(e) => {
                        ;(e.currentTarget as HTMLImageElement).style.display = "none"
                      }}
                    />
                  ) : (
                    getFileIcon(file.mime_type)
                  )}
                </div>

                {/* 文件名 */}
                <p className="text-[10px] font-medium leading-tight text-foreground/80 break-all line-clamp-2 text-center">
                  {file.file_name}
                </p>

                {/* 元数据 */}
                <div className="text-[10px] text-muted-foreground text-center space-y-0.5">
                  <p>{formatFileSize(file.file_size)}</p>
                  <p className="truncate">{formatDate(file.created_at)}</p>
                </div>

                {/* Hover 操作按钮 */}
                <div
                  className="absolute inset-x-2 bottom-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={(e) => e.stopPropagation()}
                >
                  <Button
                    size="icon"
                    variant="secondary"
                    className="h-6 w-6 rounded-md flex-1"
                    title="下载"
                    onClick={() => handleDownload(file)}
                  >
                    <Download className="size-3" />
                  </Button>
                  <Button
                    size="icon"
                    variant="secondary"
                    className="h-6 w-6 rounded-md text-destructive hover:bg-destructive/10"
                    title="删除"
                    onClick={() => setDeleteTarget(file)}
                  >
                    <Trash2 className="size-3" />
                  </Button>
                </div>
              </motion.div>
            )
          })}
        </div>
      )}

      {/* 分页 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <Button
            size="sm"
            variant="outline"
            className="border-dashed text-xs h-7 px-3"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            上一页
          </Button>
          <span className="text-xs text-muted-foreground">
            {page} / {totalPages}
          </span>
          <Button
            size="sm"
            variant="outline"
            className="border-dashed text-xs h-7 px-3"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            下一页
          </Button>
        </div>
      )}

      {/* 删除确认 Dialog */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除文件</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除文件{" "}
              <span className="font-semibold text-foreground">「{deleteTarget?.file_name}」</span>{" "}
              吗？此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
              disabled={deleteMutation.isPending}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {deleteMutation.isPending && <Loader2 className="mr-1.5 size-3.5 animate-spin" />}
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </motion.div>
  )
}
