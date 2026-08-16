'use client';

import * as React from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { motion } from 'motion/react';
import { Library, RefreshCw } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { RequireAuth } from '@/components/auth/require-auth';
import { ItemList } from '@/components/common/item/item-list';
import {
  CONTENT_TYPE_OPTIONS,
  IMPORTANCE_OPTIONS,
  LIFECYCLE_OPTIONS,
} from '@/components/common/item/labels';
import { Button } from '@/components/ui/button';
import services from '@/lib/services';
import type {
  ItemContentType,
  ItemImportance,
  ItemLifecycle,
  ListItemsParams,
} from '@/lib/services/item/types';
import { cn } from '@/lib/utils';

const LIBRARY_QUERY_KEY = ['items', 'library'] as const;

export default function LibraryPage() {
  return (
    <RequireAuth>
      <LibraryView />
    </RequireAuth>
  );
}

function LibraryView() {
  const t = useTranslations('item.library');
  const tItem = useTranslations('item');
  const tCommon = useTranslations('item.common');
  const queryClient = useQueryClient();
  const [contentType, setContentType] = React.useState<ItemContentType | null>(
    null,
  );
  const [lifecycle, setLifecycle] = React.useState<ItemLifecycle | null>(null);
  const [importance, setImportance] = React.useState<ItemImportance | null>(
    null,
  );

  const params = React.useMemo((): ListItemsParams => {
    const p: ListItemsParams = { page: 1, page_size: 50 };
    if (contentType) p.content_type = contentType;
    if (lifecycle) {
      p.lifecycle = lifecycle;
      if (lifecycle === 'archived') p.include_archived = true;
      if (lifecycle === 'trash') p.include_trash = true;
    }
    if (importance) p.importance = importance;
    return p;
  }, [contentType, lifecycle, importance]);

  const listQuery = useQuery({
    queryKey: [...LIBRARY_QUERY_KEY, params],
    queryFn: () => services.item.list(params),
  });

  React.useEffect(() => {
    if (listQuery.isError) {
      const err = listQuery.error;
      toast.error(err instanceof Error ? err.message : t('loadFailed'));
    }
  }, [listQuery.isError, listQuery.error, t]);

  const refresh = React.useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: LIBRARY_QUERY_KEY });
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
          <Library className='size-5 text-primary' />
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

      <div className='flex flex-col gap-4'>
        <ChipGroup
          title={t('contentType')}
          options={CONTENT_TYPE_OPTIONS.map((v) => ({
            value: v,
            label: tItem(`contentType.${v}`),
          }))}
          value={contentType}
          onChange={setContentType}
          allLabel={tCommon('all')}
        />
        <ChipGroup
          title={t('lifecycle')}
          options={LIFECYCLE_OPTIONS.map((v) => ({
            value: v,
            label: tItem(`lifecycle.${v}`),
          }))}
          value={lifecycle}
          onChange={setLifecycle}
          allLabel={tCommon('all')}
        />
        <ChipGroup
          title={t('importance')}
          options={IMPORTANCE_OPTIONS.map((v) => ({
            value: v,
            label: tItem(`importance.${v}`),
          }))}
          value={importance}
          onChange={setImportance}
          allLabel={tCommon('all')}
        />
      </div>

      <div className='text-xs text-muted-foreground'>
        {listQuery.isPending
          ? tCommon('loading')
          : tCommon('total', { count: listQuery.data?.total ?? 0 })}
      </div>

      <ItemList
        items={listQuery.data?.results ?? []}
        loading={listQuery.isPending}
        variant='grid'
        emptyTitle={t('emptyTitle')}
        emptyDescription={t('emptyDescription')}
        onChanged={refresh}
      />
    </motion.div>
  );
}

function ChipGroup<T extends string>({
  title,
  options,
  value,
  onChange,
  allLabel,
}: {
  title: string;
  options: { value: T; label: string }[];
  value: T | null;
  onChange: (v: T | null) => void;
  allLabel: string;
}) {
  return (
    <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-3'>
      <span className='shrink-0 text-xs font-medium text-muted-foreground w-16'>
        {title}
      </span>
      <div className='flex flex-wrap gap-2'>
        <Chip
          active={value === null}
          onClick={() => onChange(null)}
          label={allLabel}
        />
        {options.map((o) => (
          <Chip
            key={o.value}
            active={value === o.value}
            onClick={() => onChange(value === o.value ? null : o.value)}
            label={o.label}
          />
        ))}
      </div>
    </div>
  );
}

function Chip({
  active,
  onClick,
  label,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type='button'
      onClick={onClick}
      className={cn(
        'rounded-full border px-3 py-1 text-xs transition-colors',
        active
          ? 'border-primary bg-primary text-primary-foreground'
          : 'border-border bg-background text-muted-foreground hover:bg-muted/60',
      )}
    >
      {label}
    </button>
  );
}
