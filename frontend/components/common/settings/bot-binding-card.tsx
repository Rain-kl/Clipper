// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

'use client';

import * as React from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Bot, Loader2, Unlink } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { UserMessageGatewayService } from '@/lib/services/message-gateway';

export function BotBindingCard() {
  const t = useTranslations('settings.botBinding');
  const queryClient = useQueryClient();
  const [open, setOpen] = React.useState(false);
  const [channelId, setChannelId] = React.useState('');
  const [code, setCode] = React.useState('');

  const bindingsQuery = useQuery({
    queryKey: ['message-gateway', 'bindings'],
    queryFn: () => UserMessageGatewayService.listBindings(),
  });
  const channelsQuery = useQuery({
    queryKey: ['message-gateway', 'channels'],
    queryFn: () => UserMessageGatewayService.listChannels(),
    enabled: open,
  });

  const bindMutation = useMutation({
    mutationFn: () =>
      UserMessageGatewayService.bind({ channel_id: channelId, code }),
    onSuccess: () => {
      toast.success(t('bindSuccess'));
      queryClient.invalidateQueries({ queryKey: ['message-gateway', 'bindings'] });
      setOpen(false);
      setChannelId('');
      setCode('');
    },
    onError: (err: unknown) => {
      toast.error(t('bindFailed') + ': ' + (err as Error).message);
    },
  });

  const unbindMutation = useMutation({
    mutationFn: (id: string) => UserMessageGatewayService.unbind(id),
    onSuccess: () => {
      toast.success(t('unbindSuccess'));
      queryClient.invalidateQueries({ queryKey: ['message-gateway', 'bindings'] });
    },
    onError: (err: unknown) => {
      toast.error(t('unbindFailed') + ': ' + (err as Error).message);
    },
  });

  const typeLabel = (typ: string) =>
    typ === 'qq' ? t('typeQQ') : t('typeTelegram');

  return (
    <div className='space-y-6 bg-card border border-dashed rounded-lg p-6 flex flex-col'>
      <div className='border-b pb-4 flex items-center gap-2'>
        <div className='p-1.5 rounded-lg bg-primary/10 text-primary'>
          <Bot className='size-4' />
        </div>
        <div>
          <h2 className='text-base font-semibold tracking-tight'>{t('title')}</h2>
          <p className='text-[11px] text-muted-foreground'>{t('description')}</p>
        </div>
      </div>

      <div className='space-y-2'>
        <h3 className='text-[11px] font-semibold text-muted-foreground uppercase tracking-wider'>
          {t('bound')}
        </h3>
        {bindingsQuery.isPending ? (
          <div className='flex items-center justify-center py-4'>
            <Loader2 className='size-4 animate-spin text-primary' />
          </div>
        ) : bindingsQuery.isError ? (
          <p className='text-[11px] text-destructive'>{t('loadFailed')}</p>
        ) : (bindingsQuery.data ?? []).length > 0 ? (
          <div className='space-y-2'>
            {(bindingsQuery.data ?? []).map((binding) => (
              <div
                key={binding.id}
                className='flex items-center justify-between gap-4 rounded-xl border border-dashed p-3 bg-card'
              >
                <div className='space-y-0.5 min-w-0'>
                  <span className='font-semibold text-xs block truncate'>
                    {binding.channel_name || typeLabel(binding.channel_type)}
                  </span>
                  <span className='text-[10px] text-muted-foreground font-mono block truncate'>
                    {t('platformUserId')}: {binding.platform_user_id}
                  </span>
                </div>
                <Button
                  type='button'
                  variant='ghost'
                  size='sm'
                  className='text-[11px] text-muted-foreground hover:text-rose-500 h-7 px-2'
                  disabled={unbindMutation.isPending}
                  onClick={() => unbindMutation.mutate(binding.id)}
                >
                  <Unlink className='size-3 mr-1' />
                  {t('unbind')}
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <div className='rounded-xl border border-dashed px-4 py-6 text-center text-[11px] text-muted-foreground'>
            {t('empty')}
          </div>
        )}
      </div>

      <Button
        type='button'
        variant='outline'
        className='w-full border-dashed'
        onClick={() => setOpen(true)}
      >
        {t('bind')}
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('bindDialogTitle')}</DialogTitle>
            <DialogDescription>{t('bindDialogDescription')}</DialogDescription>
          </DialogHeader>
          <div className='space-y-4'>
            <div className='space-y-2'>
              <Label>{t('selectChannel')}</Label>
              <Select value={channelId} onValueChange={setChannelId}>
                <SelectTrigger>
                  <SelectValue placeholder={t('selectChannel')} />
                </SelectTrigger>
                <SelectContent>
                  {(channelsQuery.data ?? []).map((ch) => (
                    <SelectItem key={ch.id} value={ch.id}>
                      {ch.name} ({typeLabel(ch.type)})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {!channelsQuery.isPending &&
              (channelsQuery.data ?? []).length === 0 ? (
                <p className='text-xs text-muted-foreground'>{t('noChannels')}</p>
              ) : null}
            </div>
            <div className='space-y-2'>
              <Label htmlFor='mg-code'>{t('pairingCode')}</Label>
              <Input
                id='mg-code'
                value={code}
                placeholder={t('pairingCodePlaceholder')}
                autoCapitalize='characters'
                onChange={(e) => setCode(e.target.value.toUpperCase())}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setOpen(false)}>
              {t('cancel')}
            </Button>
            <Button
              disabled={
                !channelId || code.trim() === '' || bindMutation.isPending
              }
              onClick={() => bindMutation.mutate()}
            >
              {bindMutation.isPending ? (
                <Loader2 className='size-4 animate-spin' />
              ) : null}
              {t('submit')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
