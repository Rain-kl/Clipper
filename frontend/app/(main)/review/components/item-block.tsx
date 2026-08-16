'use client';

import * as React from 'react';
import {
  Archive,
  FileIcon,
  ImageIcon,
  Lock,
  MoreHorizontal,
  NotebookPen,
  Scissors,
  Trash2,
} from 'lucide-react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import services, { getFileUrl } from '@/lib/services';
import type { Item, ItemImportance } from '@/lib/services/item/types';
import { cn } from '@/lib/utils';

type ItemBlockProps = {
  item: Item;
  onChanged: () => void;
  muted?: boolean;
};

export function ItemBlock({ item, onChanged, muted = false }: ItemBlockProps) {
  const t = useTranslations('item');
  const [busy, setBusy] = React.useState(false);

  const run = React.useCallback(
    async (action: () => Promise<unknown>, okMsg: string) => {
      if (busy) return;
      setBusy(true);
      try {
        await action();
        toast.success(okMsg);
        onChanged();
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t('common.actionFailed'),
        );
      } finally {
        setBusy(false);
      }
    },
    [busy, onChanged, t],
  );

  const classify = (importance: ItemImportance) =>
    run(
      () =>
        services.item.update(item.id, {
          lifecycle: 'active',
          importance,
        }),
      t('common.classifiedAs', { label: t(`importance.${importance}`) }),
    );

  const trash = () =>
    run(() => services.item.remove(item.id), t('common.trashed'));

  const archive = () =>
    run(
      () => services.item.update(item.id, { lifecycle: 'archived' }),
      t('common.archived'),
    );

  const isPending = item.lifecycle === 'pending';
  const isActive = item.lifecycle === 'active';
  const isArchived = item.lifecycle === 'archived';

  const attachments = item.attachments ?? [];
  const imageAttachments = attachments.filter((a) => {
    if (a.mime_type?.startsWith('image/')) return true;
    if (a.mime_type) return false;
    return item.content_type === 'image';
  });
  const fileAttachments = attachments.filter(
    (a) => !imageAttachments.some((img) => img.id === a.id),
  );

  const timeLabel = formatTime(item.created_at);

  return (
    <div
      className={cn(
        'group relative rounded-lg border bg-card px-3 py-3 transition-colors',
        muted && 'opacity-70',
        isPending && 'border-primary/30 bg-primary/5',
      )}
    >
      <div className='flex items-start justify-between gap-2'>
        <div className='min-w-0 flex-1 space-y-1.5'>
          <div className='flex flex-wrap items-center gap-1.5'>
            {item.title ? (
              <span className='truncate text-sm font-medium'>{item.title}</span>
            ) : null}
            {isPending ? (
              <Badge variant='secondary'>{t('common.pendingBadge')}</Badge>
            ) : null}
            {isActive && item.importance !== 'none' ? (
              <Badge variant='outline'>
                {t(`importance.${item.importance}`)}
              </Badge>
            ) : null}
            {isArchived ? (
              <Badge variant='secondary'>{t('common.archivedBadge')}</Badge>
            ) : null}
            <span className='text-xs text-muted-foreground'>{timeLabel}</span>
          </div>

          {item.body ? (
            <p className='line-clamp-4 whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground/90'>
              {item.body}
            </p>
          ) : null}

          {imageAttachments.length > 0 ? (
            <div className='flex flex-wrap gap-2 pt-1'>
              {imageAttachments.map((att) => {
                const url = getFileUrl(att.upload_id, 'medium');
                if (!url) return null;
                return (
                  <a
                    key={att.id}
                    href={getFileUrl(att.upload_id, 'origin') ?? url}
                    target='_blank'
                    rel='noreferrer'
                    className='block overflow-hidden rounded-md border bg-muted'
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element -- dynamic upload URLs */}
                    <img
                      src={url}
                      alt={att.file_name || t('common.imageAlt')}
                      className='h-20 w-20 object-cover'
                      loading='lazy'
                    />
                  </a>
                );
              })}
            </div>
          ) : null}

          {fileAttachments.length > 0 ? (
            <ul className='space-y-1 pt-1'>
              {fileAttachments.map((att) => {
                const url = getFileUrl(att.upload_id);
                return (
                  <li key={att.id}>
                    <a
                      href={url ?? '#'}
                      target='_blank'
                      rel='noreferrer'
                      className='inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground'
                    >
                      {att.mime_type?.startsWith('image/') ? (
                        <ImageIcon className='size-3.5' />
                      ) : (
                        <FileIcon className='size-3.5' />
                      )}
                      <span className='truncate'>
                        {att.file_name ||
                          t('common.fileFallback', { id: att.upload_id })}
                      </span>
                    </a>
                  </li>
                );
              })}
            </ul>
          ) : null}

          {!item.body &&
          imageAttachments.length === 0 &&
          fileAttachments.length === 0 ? (
            <p className='text-sm text-muted-foreground'>
              {t('common.noBody')}
            </p>
          ) : null}
        </div>

        {isPending ? (
          <div
            className={cn(
              'absolute right-2 top-2 flex items-center gap-0.5 rounded-md border bg-background/95 p-0.5 shadow-sm backdrop-blur',
              'opacity-100 md:opacity-0 md:group-hover:opacity-100 md:group-focus-within:opacity-100',
              'transition-opacity',
            )}
          >
            <Button
              type='button'
              size='sm'
              variant='ghost'
              disabled={busy}
              className='h-7 px-2 text-xs'
              onClick={() => void trash()}
              title={t('common.trashAction')}
            >
              <Trash2 className='size-3.5' />
              <span className='hidden sm:inline'>
                {t('common.trashAction')}
              </span>
            </Button>
            <Button
              type='button'
              size='sm'
              variant='ghost'
              disabled={busy}
              className='h-7 px-2 text-xs'
              onClick={() => void classify('fragment')}
              title={t('importance.fragment')}
            >
              <Scissors className='size-3.5' />
              <span className='hidden sm:inline'>
                {t('importance.fragment')}
              </span>
            </Button>
            <Button
              type='button'
              size='sm'
              variant='ghost'
              disabled={busy}
              className='h-7 px-2 text-xs'
              onClick={() => void classify('note')}
              title={t('importance.note')}
            >
              <NotebookPen className='size-3.5' />
              <span className='hidden sm:inline'>{t('importance.note')}</span>
            </Button>
            <Button
              type='button'
              size='sm'
              variant='ghost'
              disabled={busy}
              className='h-7 px-2 text-xs'
              onClick={() => void classify('vault')}
              title={t('importance.vault')}
            >
              <Lock className='size-3.5' />
              <span className='hidden sm:inline'>{t('importance.vault')}</span>
            </Button>
          </div>
        ) : null}

        {(isActive || isArchived) && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type='button'
                size='icon-sm'
                variant='ghost'
                disabled={busy}
                className='shrink-0'
                aria-label={t('common.moreActions')}
              >
                <MoreHorizontal className='size-4' />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end'>
              {isActive ? (
                <>
                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>
                      {t('common.reclassify')}
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent>
                      <DropdownMenuItem
                        onClick={() => void classify('fragment')}
                      >
                        <Scissors className='size-4' />
                        {t('importance.fragment')}
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => void classify('note')}>
                        <NotebookPen className='size-4' />
                        {t('importance.note')}
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => void classify('vault')}>
                        <Lock className='size-4' />
                        {t('importance.vault')}
                      </DropdownMenuItem>
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>
                  <DropdownMenuItem onClick={() => void archive()}>
                    <Archive className='size-4' />
                    {t('common.archived')}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                </>
              ) : null}
              <DropdownMenuItem
                variant='destructive'
                onClick={() => void trash()}
              >
                <Trash2 className='size-4' />
                {t('common.moveToTrash')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </div>
  );
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
  });
}
