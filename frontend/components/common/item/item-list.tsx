'use client';

import { FileTextIcon } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { EmptyStateWithBorder } from '@/components/layout/empty';
import { LoadingStateWithBorder } from '@/components/layout/loading';
import type { Item } from '@/lib/services/item/types';
import { cn } from '@/lib/utils';

import { ItemRow } from './item-row';

type ItemListProps = {
  items: Item[];
  loading?: boolean;
  emptyTitle?: string;
  emptyDescription?: string;
  onChanged?: () => void;
  className?: string;
  variant?: 'list' | 'grid';
};

export function ItemList({
  items,
  loading,
  emptyTitle,
  emptyDescription,
  onChanged,
  className,
  variant = 'list',
}: ItemListProps) {
  const t = useTranslations('item.common');
  const resolvedTitle = emptyTitle ?? t('emptyDefaultTitle');
  const resolvedDescription = emptyDescription ?? t('emptyDefaultDescription');

  if (loading) {
    return <LoadingStateWithBorder className={className} />;
  }

  if (!items.length) {
    return (
      <EmptyStateWithBorder
        className={className}
        title={resolvedTitle}
        description={resolvedDescription}
        icon={FileTextIcon}
      />
    );
  }

  return (
    <div
      className={cn(
        variant === 'grid'
          ? 'grid gap-3 sm:grid-cols-2 xl:grid-cols-3'
          : 'flex flex-col gap-2',
        className,
      )}
    >
      {items.map((item) => (
        <ItemRow key={item.id} item={item} onChanged={onChanged} />
      ))}
    </div>
  );
}
