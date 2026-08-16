// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

'use client';

import * as React from 'react';
import { useTranslations } from 'next-intl';
import { Loader2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
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
import { TelegramForm, type TelegramFormValue } from '../channels/telegram/form';
import { QQForm, type QQFormValue } from '../channels/qq/form';
import type { CreateMessageChannelRequest } from '@/lib/services/message-gateway';

interface AddChannelDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: CreateMessageChannelRequest) => void;
  submitting?: boolean;
}

export function AddChannelDialog({
  open,
  onOpenChange,
  onSubmit,
  submitting,
}: AddChannelDialogProps) {
  const t = useTranslations('admin.messageGateway');
  const [name, setName] = React.useState('');
  const [type, setType] = React.useState<'telegram' | 'qq' | ''>('');
  const [telegram, setTelegram] = React.useState<TelegramFormValue>({
    bot_token: '',
    base_url: '',
  });
  const [qq, setQQ] = React.useState<QQFormValue>({
    app_id: '',
    app_secret: '',
    portal_host: '',
  });

  React.useEffect(() => {
    if (!open) {
      setName('');
      setType('');
      setTelegram({ bot_token: '', base_url: '' });
      setQQ({ app_id: '', app_secret: '', portal_host: '' });
    }
  }, [open]);

  const canSubmit =
    name.trim() !== '' &&
    ((type === 'telegram' && telegram.bot_token.trim() !== '') ||
      (type === 'qq' && qq.app_id.trim() !== '' && qq.app_secret.trim() !== ''));

  const handleSubmit = () => {
    if (!canSubmit || !type) return;
    const data: CreateMessageChannelRequest = {
      name: name.trim(),
      type,
      enabled: true,
    };
    if (type === 'telegram') {
      data.bot_token = telegram.bot_token.trim();
      if (telegram.base_url.trim()) data.base_url = telegram.base_url.trim();
    } else {
      data.app_id = qq.app_id.trim();
      data.app_secret = qq.app_secret.trim();
      if (qq.portal_host.trim()) data.portal_host = qq.portal_host.trim();
    }
    onSubmit(data);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('addChannel')}</DialogTitle>
        </DialogHeader>
        <div className='space-y-4'>
          <div className='space-y-2'>
            <Label htmlFor='mg-name'>{t('name')}</Label>
            <Input
              id='mg-name'
              value={name}
              placeholder={t('namePlaceholder')}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className='space-y-2'>
            <Label>{t('type')}</Label>
            <Select
              value={type}
              onValueChange={(v) => setType(v as 'telegram' | 'qq')}
            >
              <SelectTrigger>
                <SelectValue placeholder={t('selectType')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='telegram'>{t('typeTelegram')}</SelectItem>
                <SelectItem value='qq'>{t('typeQQ')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {type === 'telegram' ? (
            <TelegramForm value={telegram} onChange={setTelegram} />
          ) : null}
          {type === 'qq' ? <QQForm value={qq} onChange={setQQ} /> : null}
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('cancel')}
          </Button>
          <Button disabled={!canSubmit || submitting} onClick={handleSubmit}>
            {submitting ? <Loader2 className='size-4 animate-spin' /> : null}
            {t('create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
