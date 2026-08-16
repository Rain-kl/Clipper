'use client';

import * as React from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { motion } from 'motion/react';
import { RefreshCw, Search } from 'lucide-react';
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
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import services from '@/lib/services';
import type {
  ItemContentType,
  ItemImportance,
  ItemLifecycle,
  ListItemsParams,
} from '@/lib/services/item/types';

const SEARCH_QUERY_KEY = ['items', 'search'] as const;
const ALL = '__all__';

export default function SearchPage() {
  return (
    <RequireAuth>
      <SearchView />
    </RequireAuth>
  );
}

function SearchView() {
  const t = useTranslations('item.search');
  const tItem = useTranslations('item');
  const tCommon = useTranslations('item.common');
  const queryClient = useQueryClient();
  const [draftQ, setDraftQ] = React.useState('');
  const [q, setQ] = React.useState('');
  const [lifecycle, setLifecycle] = React.useState<string>(ALL);
  const [importance, setImportance] = React.useState<string>(ALL);
  const [contentType, setContentType] = React.useState<string>(ALL);
  const [includeArchived, setIncludeArchived] = React.useState(false);
  const [includeTrash, setIncludeTrash] = React.useState(false);

  const params = React.useMemo((): ListItemsParams => {
    const p: ListItemsParams = {
      page: 1,
      page_size: 50,
      include_archived: includeArchived,
      include_trash: includeTrash,
    };
    if (q.trim()) p.q = q.trim();
    if (lifecycle !== ALL) p.lifecycle = lifecycle as ItemLifecycle;
    if (importance !== ALL) p.importance = importance as ItemImportance;
    if (contentType !== ALL) p.content_type = contentType as ItemContentType;
    if (lifecycle === 'archived') p.include_archived = true;
    if (lifecycle === 'trash') p.include_trash = true;
    return p;
  }, [q, lifecycle, importance, contentType, includeArchived, includeTrash]);

  const listQuery = useQuery({
    queryKey: [...SEARCH_QUERY_KEY, params],
    queryFn: () => services.item.list(params),
  });

  React.useEffect(() => {
    if (listQuery.isError) {
      const err = listQuery.error;
      toast.error(err instanceof Error ? err.message : t('loadFailed'));
    }
  }, [listQuery.isError, listQuery.error, t]);

  const refresh = React.useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: SEARCH_QUERY_KEY });
  }, [queryClient]);

  const submitSearch = (e?: React.FormEvent) => {
    e?.preventDefault();
    setQ(draftQ);
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: 'easeOut' }}
      className='flex w-full flex-col gap-6 py-6 px-1'
    >
      <div className='flex items-center justify-between gap-4'>
        <div className='flex items-center gap-2'>
          <Search className='size-5 text-primary' />
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

      <form
        onSubmit={submitSearch}
        className='flex flex-col gap-4 rounded-xl border bg-card/40 p-4'
      >
        <div className='flex flex-col gap-2 sm:flex-row sm:items-center'>
          <Input
            value={draftQ}
            onChange={(e) => setDraftQ(e.target.value)}
            placeholder={t('placeholder')}
            className='flex-1'
          />
          <Button type='submit' className='sm:w-auto'>
            {t('submit')}
          </Button>
        </div>

        <div className='flex flex-wrap gap-3'>
          <FilterSelect
            label={t('lifecycle')}
            value={lifecycle}
            onChange={setLifecycle}
            allLabel={tCommon('all')}
            options={LIFECYCLE_OPTIONS.map((v) => ({
              value: v,
              label: tItem(`lifecycle.${v}`),
            }))}
          />
          <FilterSelect
            label={t('importance')}
            value={importance}
            onChange={setImportance}
            allLabel={tCommon('all')}
            options={IMPORTANCE_OPTIONS.map((v) => ({
              value: v,
              label: tItem(`importance.${v}`),
            }))}
          />
          <FilterSelect
            label={t('type')}
            value={contentType}
            onChange={setContentType}
            allLabel={tCommon('all')}
            options={CONTENT_TYPE_OPTIONS.map((v) => ({
              value: v,
              label: tItem(`contentType.${v}`),
            }))}
          />
        </div>

        <div className='flex flex-wrap gap-6'>
          <label className='flex items-center gap-2 text-sm text-muted-foreground'>
            <Checkbox
              checked={includeArchived}
              onCheckedChange={(v) => setIncludeArchived(v === true)}
            />
            {t('includeArchived')}
          </label>
          <label className='flex items-center gap-2 text-sm text-muted-foreground'>
            <Checkbox
              checked={includeTrash}
              onCheckedChange={(v) => setIncludeTrash(v === true)}
            />
            {t('includeTrash')}
          </label>
        </div>
      </form>

      <div className='text-xs text-muted-foreground'>
        {listQuery.isPending
          ? tCommon('loading')
          : tCommon('total', { count: listQuery.data?.total ?? 0 })}
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

function FilterSelect({
  label,
  value,
  onChange,
  options,
  allLabel,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: { value: string; label: string }[];
  allLabel: string;
}) {
  return (
    <div className='flex flex-col gap-1.5'>
      <Label className='text-xs text-muted-foreground'>{label}</Label>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className='w-[140px] text-xs' size='sm'>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>{allLabel}</SelectItem>
          {options.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
