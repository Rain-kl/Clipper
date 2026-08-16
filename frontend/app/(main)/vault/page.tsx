'use client';

import * as React from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { motion } from 'motion/react';
import { Lock, RefreshCw } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { RequireAuth } from '@/components/auth/require-auth';
import { ItemList } from '@/components/common/item/item-list';
import { Button } from '@/components/ui/button';
import services from '@/lib/services';

const VAULT_QUERY_KEY = ['items', 'vault'] as const;

export default function VaultPage() {
  return (
    <RequireAuth>
      <VaultView />
    </RequireAuth>
  );
}

function VaultView() {
  const t = useTranslations('item.vault');
  const tCommon = useTranslations('item.common');
  const queryClient = useQueryClient();

  const listQuery = useQuery({
    queryKey: VAULT_QUERY_KEY,
    queryFn: () =>
      services.item.list({
        page: 1,
        page_size: 50,
        importance: 'vault',
        include_archived: false,
        include_trash: false,
      }),
  });

  React.useEffect(() => {
    if (listQuery.isError) {
      const err = listQuery.error;
      toast.error(err instanceof Error ? err.message : t('loadFailed'));
    }
  }, [listQuery.isError, listQuery.error, t]);

  const refresh = React.useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: VAULT_QUERY_KEY });
  }, [queryClient]);

  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: 'easeOut' }}
      className='flex w-full flex-col gap-6 py-6 px-1'
    >
      <div className='flex items-center justify-between gap-4'>
        <div className='flex items-center gap-2'>
          <Lock className='size-5 text-primary' />
          <h1 className='text-2xl font-semibold tracking-tight'>
            {t('title')}
          </h1>
        </div>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={listQuery.isFetching}
          onClick={() => void listQuery.refetch()}
        >
          <RefreshCw
            className={listQuery.isFetching ? 'size-4 animate-spin' : 'size-4'}
          />
          {tCommon('refresh')}
        </Button>
      </div>

      <div className='rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-950 dark:text-amber-100'>
        {t('warning')}
      </div>

      <div className='text-xs text-muted-foreground'>
        {listQuery.isPending
          ? tCommon('loading')
          : t('total', { count: listQuery.data?.total ?? 0 })}
      </div>

      <ItemList
        items={listQuery.data?.results ?? []}
        loading={listQuery.isPending}
        emptyTitle={t('emptyTitle')}
        emptyDescription={t('emptyDescription')}
        onChanged={refresh}
      />
    </motion.div>
  );
}
