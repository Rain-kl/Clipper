'use client';

import * as React from 'react';
import {
  Archive,
  FileIcon,
  ImageIcon,
  MoreHorizontal,
  RotateCcw,
  StickyNote,
  Trash2,
  Type,
} from 'lucide-react';
import { useLocale, useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { FileImagePreview } from '@/components/common/file-image-preview';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import type { AppLocale } from '@/i18n/config';
import { formatDateTime } from '@/i18n/format';
import services from '@/lib/services';
import type {
  Item,
  ItemImportance,
  PatchItemPayload,
} from '@/lib/services/item/types';
import { cn } from '@/lib/utils';

type ItemRowProps = {
  item: Item;
  onChanged?: () => void;
  className?: string;
};

export function ItemRow({ item, onChanged, className }: ItemRowProps) {
  const t = useTranslations('item');
  const locale = useLocale() as AppLocale;
  const [busy, setBusy] = React.useState(false);
  const title = item.title?.trim();
  const body = item.body?.trim();
  const previewTitle = title || (!body ? t('common.noContent') : '');
  const previewBody = body || '';
  const imageAtt = item.attachments?.find(
    (a) => a.mime_type?.startsWith('image/') || item.content_type === 'image',
  );

  const run = async (fn: () => Promise<unknown>, okMsg: string) => {
    if (busy) return;
    setBusy(true);
    try {
      await fn();
      toast.success(okMsg);
      onChanged?.();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('common.actionFailed'));
    } finally {
      setBusy(false);
    }
  };

  const classify = (importance: ItemImportance) =>
    run(
      () =>
        services.item.update(item.id, {
          lifecycle: 'active',
          importance,
        }),
      t('common.markedAs', { label: t(`importance.${importance}`) }),
    );

  const patch = (payload: PatchItemPayload, okMsg: string) =>
    run(() => services.item.update(item.id, payload), okMsg);

  const trash = () =>
    run(() => services.item.remove(item.id), t('common.trashedBin'));

  const restore = () =>
    patch(
      {
        lifecycle: item.importance === 'none' ? 'pending' : 'active',
      },
      t('common.restored'),
    );

  const archive = () =>
    patch({ lifecycle: 'archived' }, t('common.archived'));

  const showPendingActions =
    item.lifecycle === 'pending' || item.lifecycle === 'active';
  const showTrashRestore = item.lifecycle === 'trash';
  const showArchiveRestore = item.lifecycle === 'archived';

  return (
    <div
      className={cn(
        'group relative flex gap-3 rounded-xl border bg-card p-3 shadow-sm transition-colors hover:bg-muted/30',
        className,
      )}
    >
      <div className='flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-muted'>
        {imageAtt?.upload_id ? (
          <FileImagePreview
            fileId={imageAtt.upload_id}
            alt={title || t('common.image')}
            quality='low'
            variant='compact'
            className='size-full object-cover'
          />
        ) : item.content_type === 'image' ? (
          <ImageIcon className='size-5 text-muted-foreground' />
        ) : item.content_type === 'file' ? (
          <FileIcon className='size-5 text-muted-foreground' />
        ) : (
          <Type className='size-5 text-muted-foreground' />
        )}
      </div>

      <div className='min-w-0 flex-1 space-y-1.5'>
        <div className='flex flex-wrap items-center gap-1.5'>
          <Badge variant='outline' className='text-[10px]'>
            {t(`contentType.${item.content_type}`)}
          </Badge>
          <Badge variant='secondary' className='text-[10px]'>
            {t(`lifecycle.${item.lifecycle}`)}
          </Badge>
          {item.importance !== 'none' && (
            <Badge variant='default' className='text-[10px]'>
              {t(`importance.${item.importance}`)}
            </Badge>
          )}
          <span className='ml-auto text-[11px] text-muted-foreground'>
            {formatDateTime(item.created_at, locale)}
          </span>
        </div>

        {previewTitle ? (
          <p className='truncate text-sm font-medium'>{previewTitle}</p>
        ) : null}
        {previewBody ? (
          <p className='line-clamp-2 text-sm text-muted-foreground whitespace-pre-wrap'>
            {previewBody}
          </p>
        ) : null}

        {showPendingActions && (
          <div className='flex flex-wrap items-center gap-1 pt-0.5 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100 transition-opacity'>
            <Button
              type='button'
              size='sm'
              variant='ghost'
              className='h-7 px-2 text-xs'
              disabled={busy}
              onClick={() => void trash()}
            >
              <Trash2 className='size-3.5' />
              {t('common.trashAction')}
            </Button>
            <Button
              type='button'
              size='sm'
              variant='ghost'
              className='h-7 px-2 text-xs'
              disabled={busy}
              onClick={() => void classify('fragment')}
            >
              {t('importance.fragment')}
            </Button>
            <Button
              type='button'
              size='sm'
              variant='ghost'
              className='h-7 px-2 text-xs'
              disabled={busy}
              onClick={() => void classify('note')}
            >
              <StickyNote className='size-3.5' />
              {t('importance.note')}
            </Button>
            <Button
              type='button'
              size='sm'
              variant='ghost'
              className='h-7 px-2 text-xs'
              disabled={busy}
              onClick={() => void classify('vault')}
            >
              {t('importance.vault')}
            </Button>
            {item.lifecycle === 'active' && (
              <Button
                type='button'
                size='sm'
                variant='ghost'
                className='h-7 px-2 text-xs'
                disabled={busy}
                onClick={() => void archive()}
              >
                <Archive className='size-3.5' />
                {t('common.archived')}
              </Button>
            )}
          </div>
        )}
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            type='button'
            variant='ghost'
            size='icon'
            className='size-8 shrink-0'
            disabled={busy}
            aria-label={t('common.moreActions')}
          >
            <MoreHorizontal className='size-4' />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end' className='w-40'>
          {showPendingActions && (
            <>
              <DropdownMenuItem onClick={() => void classify('fragment')}>
                {t('importance.fragment')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => void classify('note')}>
                {t('importance.note')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => void classify('vault')}>
                {t('importance.vault')}
              </DropdownMenuItem>
              {item.lifecycle === 'active' && (
                <DropdownMenuItem onClick={() => void archive()}>
                  <Archive className='size-3.5' />
                  {t('common.archived')}
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className='text-destructive focus:text-destructive'
                onClick={() => void trash()}
              >
                <Trash2 className='size-3.5' />
                {t('common.moveToTrashBin')}
              </DropdownMenuItem>
            </>
          )}
          {showArchiveRestore && (
            <>
              <DropdownMenuItem onClick={() => void restore()}>
                <RotateCcw className='size-3.5' />
                {t('common.restore')}
              </DropdownMenuItem>
              <DropdownMenuItem
                className='text-destructive focus:text-destructive'
                onClick={() => void trash()}
              >
                <Trash2 className='size-3.5' />
                {t('common.moveToTrashBin')}
              </DropdownMenuItem>
            </>
          )}
          {showTrashRestore && (
            <>
              <DropdownMenuItem onClick={() => void restore()}>
                <RotateCcw className='size-3.5' />
                {t('common.restore')}
              </DropdownMenuItem>
              <DropdownMenuItem
                className='text-destructive focus:text-destructive'
                onClick={() =>
                  void run(
                    () => services.item.remove(item.id, true),
                    t('common.deletedForever'),
                  )
                }
              >
                <Trash2 className='size-3.5' />
                {t('common.deleteForever')}
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
