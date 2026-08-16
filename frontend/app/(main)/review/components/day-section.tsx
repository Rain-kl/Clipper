'use client';

import * as React from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import services from '@/lib/services';
import type { Item, TimelineDay } from '@/lib/services/item/types';

import { ItemBlock } from './item-block';

type DaySectionProps = {
  day: TimelineDay;
  onChanged: () => void;
};

export function DaySection({ day, onChanged }: DaySectionProps) {
  const t = useTranslations('item.review');
  const [expanded, setExpanded] = React.useState(false);
  const [archivedItems, setArchivedItems] = React.useState<Item[]>(
    day.archived_items ?? [],
  );
  const [loadingArchived, setLoadingArchived] = React.useState(false);

  React.useEffect(() => {
    setArchivedItems(day.archived_items ?? []);
    if (!day.archived_items?.length) {
      setExpanded(false);
    }
  }, [day.date, day.archived_items, day.archived_count]);

  const label = formatDayLabel(day.date, t('today'), t('yesterday'));

  const toggleArchived = async () => {
    if (expanded) {
      setExpanded(false);
      return;
    }
    if (archivedItems.length > 0) {
      setExpanded(true);
      return;
    }
    setLoadingArchived(true);
    try {
      const res = await services.item.timeline({
        expand_archived: true,
      });
      const allArchived = res.days.flatMap((d) => d.archived_items ?? []);
      const forLocalDay = allArchived.filter(
        (item) => localDateKey(item.created_at) === day.date,
      );
      setArchivedItems(forLocalDay);
      setExpanded(true);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('loadArchivedFailed'));
    } finally {
      setLoadingArchived(false);
    }
  };

  const hasContent =
    day.items.length > 0 || day.archived_count > 0 || archivedItems.length > 0;

  if (!hasContent) return null;

  return (
    <section className='space-y-3'>
      <header className='flex items-baseline gap-2 border-b pb-2'>
        <h2 className='text-base font-semibold tracking-tight'>{label}</h2>
        <span className='text-xs text-muted-foreground'>{day.date}</span>
      </header>

      <div className='space-y-2'>
        {day.items.map((item) => (
          <ItemBlock key={item.id} item={item} onChanged={onChanged} />
        ))}
      </div>

      {day.archived_count > 0 ? (
        <div className='space-y-2 pt-1'>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            className='h-8 w-full justify-center text-muted-foreground'
            disabled={loadingArchived}
            onClick={() => void toggleArchived()}
          >
            {loadingArchived ? (
              <Spinner className='size-3.5' />
            ) : expanded ? (
              <ChevronUp className='size-3.5' />
            ) : (
              <ChevronDown className='size-3.5' />
            )}
            {expanded
              ? t('collapseArchived', { count: day.archived_count })
              : t('expandArchived', { count: day.archived_count })}
          </Button>

          {expanded && archivedItems.length > 0 ? (
            <div className='space-y-2 border-l-2 border-muted pl-3'>
              {archivedItems.map((item) => (
                <ItemBlock
                  key={item.id}
                  item={item}
                  onChanged={onChanged}
                  muted
                />
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

function formatDayLabel(
  dateStr: string,
  todayLabel: string,
  yesterdayLabel: string,
): string {
  const parts = dateStr.split('-').map(Number);
  if (parts.length !== 3 || parts.some((n) => Number.isNaN(n))) {
    return dateStr;
  }
  const [y, m, d] = parts;
  const target = new Date(y, m - 1, d);
  const today = startOfLocalDay(new Date());
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (isSameLocalDay(target, today)) return todayLabel;
  if (isSameLocalDay(target, yesterday)) return yesterdayLabel;

  return target.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'short',
  });
}

function startOfLocalDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function isSameLocalDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

function localDateKey(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}
