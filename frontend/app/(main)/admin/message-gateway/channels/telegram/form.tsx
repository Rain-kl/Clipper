// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

'use client';

import { useTranslations } from 'next-intl';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

export interface TelegramFormValue {
  bot_token: string;
  base_url: string;
}

interface TelegramFormProps {
  value: TelegramFormValue;
  onChange: (value: TelegramFormValue) => void;
  keepSecretHint?: boolean;
}

export function TelegramForm({
  value,
  onChange,
  keepSecretHint,
}: TelegramFormProps) {
  const t = useTranslations('admin.messageGateway');
  return (
    <div className='space-y-4'>
      <div className='space-y-2'>
        <Label htmlFor='mg-bot-token'>{t('botToken')}</Label>
        <Input
          id='mg-bot-token'
          type='password'
          autoComplete='off'
          value={value.bot_token}
          placeholder={
            keepSecretHint ? t('keepSecretHint') : t('botTokenPlaceholder')
          }
          onChange={(e) => onChange({ ...value, bot_token: e.target.value })}
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='mg-base-url'>{t('baseUrl')}</Label>
        <Input
          id='mg-base-url'
          value={value.base_url}
          placeholder={t('baseUrlPlaceholder')}
          onChange={(e) => onChange({ ...value, base_url: e.target.value })}
        />
      </div>
    </div>
  );
}
