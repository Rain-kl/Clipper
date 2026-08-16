'use client';

import * as React from 'react';
import { motion } from 'motion/react';
import { ClipboardPaste } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { RequireAuth } from '@/components/auth/require-auth';
import { ClipComposer } from './clip/components/composer';

export default function ClipPage() {
  const t = useTranslations('item.clip');

  return (
    <RequireAuth>
      <motion.div
        initial={{ opacity: 0, y: 15 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.35, ease: 'easeOut' }}
        className='flex w-full flex-col gap-6 py-6'
      >
        <div className='flex items-center gap-2'>
          <ClipboardPaste className='size-5 text-primary' />
          <h1 className='text-2xl font-semibold tracking-tight'>
            {t('title')}
          </h1>
        </div>

        <ClipComposer />
      </motion.div>
    </RequireAuth>
  );
}
