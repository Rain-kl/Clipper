"use client"

import * as React from "react"
import Link from "next/link"
import { ShieldCheck } from "lucide-react"

import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb"

export function SecurityMain() {
  return (
    <div className="py-6 space-y-6">
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
              <BreadcrumbPage className="text-base font-semibold">安全设置</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      </div>

      <div className="space-y-6">
        <div className="font-medium text-sm text-muted-foreground flex items-center gap-2">
          <ShieldCheck className="w-4 h-4" />
          安全设置
        </div>

        <div className="text-sm text-muted-foreground">
          当前暂无安全配置选项
        </div>
      </div>
    </div>
  )
}
