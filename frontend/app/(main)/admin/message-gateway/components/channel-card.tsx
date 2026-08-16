// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

'use client';

import * as React from 'react';
import { useTranslations } from 'next-intl';
import { Loader2, Trash2 } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import type { MessageChannel } from '@/lib/services/message-gateway';

interface ChannelCardProps {
  channel: MessageChannel;
  onToggle: (enabled: boolean) => void;
  onDelete: () => void;
  toggling?: boolean;
  deleting?: boolean;
}

export function ChannelCard({
  channel,
  onToggle,
  onDelete,
  toggling,
  deleting,
}: ChannelCardProps) {
  const t = useTranslations('admin.messageGateway');
  const [confirmOpen, setConfirmOpen] = React.useState(false);
  const typeLabel =
    channel.type === 'qq' ? t('typeQQ') : t('typeTelegram');

  return (
    <div className='rounded-xl border bg-card p-4 space-y-4'>
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <h2 className='font-medium truncate'>{channel.name}</h2>
          <Badge variant='secondary' className='mt-1'>
            {typeLabel}
          </Badge>
        </div>
        <div className='flex items-center gap-2 shrink-0'>
          <span className='text-xs text-muted-foreground'>
            {channel.enabled ? t('enabled') : t('disabled')}
          </span>
          <Switch
            checked={channel.enabled}
            disabled={toggling}
            onCheckedChange={onToggle}
          />
        </div>
      </div>
      <div className='flex justify-end'>
        <Button
          variant='ghost'
          size='sm'
          className='text-destructive'
          disabled={deleting}
          onClick={() => setConfirmOpen(true)}
        >
          {deleting ? (
            <Loader2 className='size-4 animate-spin' />
          ) : (
            <Trash2 className='size-4' />
          )}
          {t('delete')}
        </Button>
      </div>
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteConfirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('deleteConfirmDescription')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setConfirmOpen(false);
                onDelete();
              }}
            >
              {t('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
