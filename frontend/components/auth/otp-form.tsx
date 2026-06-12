"use client"

import * as React from "react"
import {RefreshCwIcon} from "lucide-react"

import {Button} from "@/components/ui/button"
import {Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle,} from "@/components/ui/card"
import {Field, FieldLabel} from "@/components/ui/field"
import {InputOTP, InputOTPGroup, InputOTPSeparator, InputOTPSlot,} from "@/components/ui/input-otp"
import {cn} from "@/lib/utils"

interface OTPFormProps {
  code: string
  setCode: (val: string) => void
  loginCodeTip: React.ReactNode
  loginCooldown: number
  isPending: boolean
  onResend: () => void
  onSubmit: () => void
}

export function OTPForm({
  code,
  setCode,
  loginCodeTip,
  loginCooldown,
  isPending,
  onResend,
  onSubmit,
}: OTPFormProps) {
  return (
    <Card className="w-full border-border/60 bg-background/80 shadow-2xl backdrop-blur">
      <CardHeader className="space-y-1.5 p-5 sm:p-6 pb-2 text-center">
        <CardTitle className="text-xl font-semibold tracking-tight text-foreground">
          验证登录
        </CardTitle>
        <CardDescription className="text-sm text-muted-foreground leading-normal">
          {loginCodeTip}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-5 p-5 sm:p-6 pt-2">
        <Field className="space-y-3">
          <div className="flex items-center justify-between">
            <FieldLabel htmlFor="otp-verification" className="text-sm font-medium">
              验证码
            </FieldLabel>
            <Button
              variant="outline"
              size="sm"
              type="button"
              onClick={onResend}
              disabled={loginCooldown > 0 || isPending}
              className="gap-1 text-xs px-2 py-1 h-7"
            >
              <RefreshCwIcon className={cn("size-3", isPending && "animate-spin")} />
              {loginCooldown > 0 ? `${loginCooldown}秒后重发` : "重新发送"}
            </Button>
          </div>
          <div className="flex justify-center">
            <InputOTP
              maxLength={6}
              id="otp-verification"
              required
              value={code}
              onChange={setCode}
              onComplete={onSubmit}
              disabled={isPending}
            >
              <InputOTPGroup className="*:data-[slot=input-otp-slot]:h-12 *:data-[slot=input-otp-slot]:w-11 *:data-[slot=input-otp-slot]:text-xl">
                <InputOTPSlot index={0} />
                <InputOTPSlot index={1} />
                <InputOTPSlot index={2} />
              </InputOTPGroup>
              <InputOTPSeparator className="mx-2" />
              <InputOTPGroup className="*:data-[slot=input-otp-slot]:h-12 *:data-[slot=input-otp-slot]:w-11 *:data-[slot=input-otp-slot]:text-xl">
                <InputOTPSlot index={3} />
                <InputOTPSlot index={4} />
                <InputOTPSlot index={5} />
              </InputOTPGroup>
            </InputOTP>
          </div>
        </Field>
      </CardContent>
      <CardFooter className="flex-col gap-4 p-5 sm:p-6 pt-2">
        <Button
          type="button"
          className="w-full"
          onClick={onSubmit}
          disabled={isPending || code.length < 6}
        >
          {isPending ? "验证中..." : "验证"}
        </Button>
        <div className="text-xs text-muted-foreground text-center">
          遇到登录问题？{" "}
          <a
            href="#"
            className="underline underline-offset-4 transition-colors hover:text-primary"
          >
            联系客服
          </a>
        </div>
      </CardFooter>
    </Card>
  )
}
