'use client';

import * as React from 'react';
import { Paperclip, Send, X } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import services from '@/lib/services';

type PendingFile = {
  id: string;
  file: File;
};

function makePendingId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

export function ClipComposer() {
  const t = useTranslations('item.clip');
  const tCommon = useTranslations('item.common');
  const [body, setBody] = React.useState('');
  const [pending, setPending] = React.useState<PendingFile[]>([]);
  const [submitting, setSubmitting] = React.useState(false);
  const fileInputRef = React.useRef<HTMLInputElement>(null);
  const textareaRef = React.useRef<HTMLTextAreaElement>(null);

  const addFiles = React.useCallback((files: FileList | File[]) => {
    const list = Array.from(files);
    if (list.length === 0) return;
    setPending((prev) => [
      ...prev,
      ...list.map((file) => ({ id: makePendingId(), file })),
    ]);
  }, []);

  const removePending = React.useCallback((id: string) => {
    setPending((prev) => prev.filter((p) => p.id !== id));
  }, []);

  const handlePaste = React.useCallback(
    (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
      const items = e.clipboardData?.items;
      if (!items) return;

      const files: File[] = [];
      for (const item of Array.from(items)) {
        if (item.kind === 'file') {
          const file = item.getAsFile();
          if (file) files.push(file);
        }
      }
      if (files.length > 0) {
        e.preventDefault();
        addFiles(files);
      }
    },
    [addFiles],
  );

  const handleSubmit = async () => {
    const text = body.trim();
    if (!text && pending.length === 0) {
      toast.error(t('needContent'));
      return;
    }

    setSubmitting(true);
    try {
      const uploadIds: string[] = [];
      for (const { file } of pending) {
        const uploaded = await services.upload.uploadFile(file, 'clip');
        uploadIds.push(uploaded.id);
      }

      await services.item.create({
        body: text || undefined,
        upload_ids: uploadIds.length > 0 ? uploadIds : undefined,
      });

      setBody('');
      setPending([]);
      if (fileInputRef.current) fileInputRef.current.value = '';
      toast.success(t('captured'));
      textareaRef.current?.focus();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('submitFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      void handleSubmit();
    }
  };

  return (
    <div className='flex w-full flex-col gap-4'>
      <Textarea
        ref={textareaRef}
        value={body}
        onChange={(e) => setBody(e.target.value)}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        placeholder={t('placeholder')}
        className='min-h-40 text-sm'
        disabled={submitting}
      />

      {pending.length > 0 && (
        <ul className='flex flex-wrap gap-2'>
          {pending.map(({ id, file }) => (
            <li
              key={id}
              className='flex items-center gap-2 rounded-md border bg-muted/40 px-2 py-1 text-xs'
            >
              <span className='max-w-48 truncate' title={file.name}>
                {file.name || tCommon('untitledFile')}
              </span>
              <span className='text-muted-foreground'>
                {(file.size / 1024).toFixed(1)} KB
              </span>
              <button
                type='button'
                className='text-muted-foreground hover:text-foreground'
                onClick={() => removePending(id)}
                disabled={submitting}
                aria-label={tCommon('removeAttachment')}
              >
                <X className='size-3.5' />
              </button>
            </li>
          ))}
        </ul>
      )}

      <div className='flex items-center justify-between gap-2'>
        <div>
          <input
            ref={fileInputRef}
            type='file'
            multiple
            className='hidden'
            onChange={(e) => {
              if (e.target.files) addFiles(e.target.files);
              e.target.value = '';
            }}
          />
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={submitting}
            onClick={() => fileInputRef.current?.click()}
          >
            <Paperclip className='size-4' />
            {t('attach')}
          </Button>
        </div>

        <Button
          type='button'
          size='sm'
          disabled={submitting}
          onClick={() => void handleSubmit()}
        >
          <Send className='size-4' />
          {submitting ? t('submitting') : t('submit')}
        </Button>
      </div>
    </div>
  );
}
