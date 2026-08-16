'use client';

import * as React from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { History, RefreshCw } from 'lucide-react';
import { motion } from 'motion/react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { RequireAuth } from '@/components/auth/require-auth';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import services from '@/lib/services';
import type { Item, TimelineDay } from '@/lib/services/item/types';

import { DaySection } from './components/day-section';

const TIMELINE_QUERY_KEY = ['items', 'timeline'] as const;

export default function ReviewPage() {
  return (
    <RequireAuth>
      <ReviewTimeline />
    </RequireAuth>
  );
}

function ReviewTimeline() {
  const t = useTranslations('item.review');
  const tCommon = useTranslations('item.common');
  const queryClient = useQueryClient();

  const timelineQuery = useQuery({
    queryKey: TIMELINE_QUERY_KEY,
    queryFn: () => services.item.timeline(),
  });

  const days = React.useMemo(
    () => regroupToLocalDays(timelineQuery.data?.days ?? []),
    [timelineQuery.data?.days],
  );

  const refresh = React.useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: TIMELINE_QUERY_KEY });
  }, [queryClient]);

  React.useEffect(() => {
    if (timelineQuery.isError) {
      const err = timelineQuery.error;
      toast.error(err instanceof Error ? err.message : t('loadFailed'));
    }
  }, [timelineQuery.isError, timelineQuery.error, t]);

  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: 'easeOut' }}
      className='flex w-full flex-col gap-6 py-6 px-1'
    >
      <div className='flex items-center justify-between gap-4'>
        <div className='flex items-center gap-2'>
          <History className='size-5 text-primary' />
          <h1 className='text-2xl font-semibold tracking-tight'>
            {t('title')}
          </h1>
        </div>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={timelineQuery.isFetching}
          onClick={() => void timelineQuery.refetch()}
        >
          <RefreshCw
            className={
              timelineQuery.isFetching ? 'size-4 animate-spin' : 'size-4'
            }
          />
          {tCommon('refresh')}
        </Button>
      </div>

      {timelineQuery.isPending ? (
        <div className='space-y-4'>
          <Skeleton className='h-8 w-40' />
          <Skeleton className='h-24 w-full' />
          <Skeleton className='h-24 w-full' />
        </div>
      ) : days.length === 0 ? (
        <div className='rounded-lg border border-dashed px-6 py-16 text-center text-sm text-muted-foreground'>
          {t('empty')}
        </div>
      ) : (
        <div className='flex flex-col gap-8'>
          {days.map((day) => (
            <DaySection key={day.date} day={day} onChanged={refresh} />
          ))}
        </div>
      )}
    </motion.div>
  );
}

/** Re-bucket server UTC day groups into the viewer's local calendar days. */
function regroupToLocalDays(serverDays: TimelineDay[]): TimelineDay[] {
  type Bucket = {
    items: Item[];
    archivedCount: number;
    archivedItems: Item[];
  };
  const map = new Map<string, Bucket>();
  const order: string[] = [];

  const ensure = (key: string): Bucket => {
    let b = map.get(key);
    if (!b) {
      b = { items: [], archivedCount: 0, archivedItems: [] };
      map.set(key, b);
      order.push(key);
    }
    return b;
  };

  for (const day of serverDays) {
    for (const item of day.items) {
      const key = localDateKey(item.created_at) || day.date;
      ensure(key).items.push(item);
    }
    for (const item of day.archived_items ?? []) {
      const key = localDateKey(item.created_at) || day.date;
      const b = ensure(key);
      b.archivedItems.push(item);
      b.archivedCount += 1;
    }
    if (
      day.archived_count > 0 &&
      (!day.archived_items || day.archived_items.length === 0)
    ) {
      const key = localDateKeyFromYMD(day.date) || day.date;
      ensure(key).archivedCount += day.archived_count;
    }
  }

  order.sort((a, b) => (a < b ? 1 : a > b ? -1 : 0));

  return order.map((date) => {
    const b = map.get(date)!;
    b.items.sort(
      (a, c) =>
        new Date(c.created_at).getTime() - new Date(a.created_at).getTime(),
    );
    b.archivedItems.sort(
      (a, c) =>
        new Date(c.created_at).getTime() - new Date(a.created_at).getTime(),
    );
    return {
      date,
      items: b.items,
      archived_count: b.archivedCount,
      archived_items: b.archivedItems.length > 0 ? b.archivedItems : undefined,
    };
  });
}

function localDateKey(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

/** Interpret a YYYY-MM-DD as a UTC calendar date and map to local day key. */
function localDateKeyFromYMD(ymd: string): string {
  const parts = ymd.split('-').map(Number);
  if (parts.length !== 3 || parts.some((n) => Number.isNaN(n))) return '';
  const [y, m, d] = parts;
  const utc = new Date(Date.UTC(y, m - 1, d));
  return localDateKey(utc.toISOString());
}
