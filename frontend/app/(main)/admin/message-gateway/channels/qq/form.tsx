// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

'use client';

import { useTranslations } from 'next-intl';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

export interface QQFormValue {
  app_id: string;
  app_secret: string;
  portal_host: string;
}

interface QQFormProps {
  value: QQFormValue;
  onChange: (value: QQFormValue) => void;
  keepSecretHint?: boolean;
}

export function QQForm({ value, onChange, keepSecretHint }: QQFormProps) {
  const t = useTranslations('admin.messageGateway');
  return (
    <div className='space-y-4'>
      <div className='space-y-2'>
        <Label htmlFor='mg-app-id'>{t('appId')}</Label>
        <Input
          id='mg-app-id'
          value={value.app_id}
          onChange={(e) => onChange({ ...value, app_id: e.target.value })}
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='mg-app-secret'>{t('appSecret')}</Label>
        <Input
          id='mg-app-secret'
          type='password'
          autoComplete='off'
          value={value.app_secret}
          placeholder={keepSecretHint ? t('keepSecretHint') : undefined}
          onChange={(e) => onChange({ ...value, app_secret: e.target.value })}
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='mg-portal-host'>{t('portalHost')}</Label>
        <Input
          id='mg-portal-host'
          value={value.portal_host}
          placeholder={t('portalHostPlaceholder')}
          onChange={(e) => onChange({ ...value, portal_host: e.target.value })}
        />
      </div>
    </div>
  );
}
