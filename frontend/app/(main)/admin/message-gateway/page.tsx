// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

'use client';

import * as React from 'react';
import { motion } from 'motion/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { MessagesSquare, Plus } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { ErrorInline } from '@/components/layout/error';
import { LoadingStateWithBorder } from '@/components/layout/loading';
import { AdminMessageGatewayService } from '@/lib/services/message-gateway';
import type { CreateMessageChannelRequest } from '@/lib/services/message-gateway';
import { AddChannelDialog } from './components/add-channel-dialog';
import { ChannelCard } from './components/channel-card';

export default function MessageGatewayAdminPage() {
  const t = useTranslations('admin.messageGateway');
  const queryClient = useQueryClient();
  const [addOpen, setAddOpen] = React.useState(false);

  const channelsQuery = useQuery({
    queryKey: ['admin', 'message-gateway-channels'],
    queryFn: () => AdminMessageGatewayService.list(),
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateMessageChannelRequest) =>
      AdminMessageGatewayService.create(data),
    onSuccess: () => {
      toast.success(t('createSuccess'));
      queryClient.invalidateQueries({
        queryKey: ['admin', 'message-gateway-channels'],
      });
      setAddOpen(false);
    },
    onError: (err: unknown) => {
      toast.error(t('createFailed') + ': ' + (err as Error).message);
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      AdminMessageGatewayService.update(id, { enabled }),
    onSuccess: () => {
      toast.success(t('updateSuccess'));
      queryClient.invalidateQueries({
        queryKey: ['admin', 'message-gateway-channels'],
      });
    },
    onError: (err: unknown) => {
      toast.error(t('updateFailed') + ': ' + (err as Error).message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => AdminMessageGatewayService.remove(id),
    onSuccess: () => {
      toast.success(t('deleteSuccess'));
      queryClient.invalidateQueries({
        queryKey: ['admin', 'message-gateway-channels'],
      });
    },
    onError: (err: unknown) => {
      toast.error(t('deleteFailed') + ': ' + (err as Error).message);
    },
  });

  const channels = channelsQuery.data ?? [];

  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: 'easeOut' }}
      className='w-full py-6 space-y-6'
    >
      <div className='flex items-start justify-between gap-4'>
        <div className='flex items-start gap-2'>
          <MessagesSquare className='size-5 text-primary mt-1' />
          <div>
            <h1 className='text-2xl font-semibold tracking-tight'>
              {t('pageTitle')}
            </h1>
            <p className='text-sm text-muted-foreground mt-1'>
              {t('pageDescription')}
            </p>
          </div>
        </div>
        <Button onClick={() => setAddOpen(true)}>
          <Plus className='size-4' />
          {t('addChannel')}
        </Button>
      </div>

      {channelsQuery.isLoading ? (
        <LoadingStateWithBorder />
      ) : channelsQuery.isError ? (
        <ErrorInline message={t('loadFailed')} />
      ) : channels.length === 0 ? (
        <div className='rounded-xl border border-dashed p-10 text-center space-y-2'>
          <h2 className='font-medium'>{t('emptyTitle')}</h2>
          <p className='text-sm text-muted-foreground'>
            {t('emptyDescription')}
          </p>
          <Button variant='outline' onClick={() => setAddOpen(true)}>
            <Plus className='size-4' />
            {t('addChannel')}
          </Button>
        </div>
      ) : (
        <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3'>
          {channels.map((ch) => (
            <ChannelCard
              key={ch.id}
              channel={ch}
              toggling={toggleMutation.isPending}
              deleting={
                deleteMutation.isPending && deleteMutation.variables === ch.id
              }
              onToggle={(enabled) =>
                toggleMutation.mutate({ id: ch.id, enabled })
              }
              onDelete={() => deleteMutation.mutate(ch.id)}
            />
          ))}
        </div>
      )}

      <AddChannelDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        submitting={createMutation.isPending}
        onSubmit={(data) => createMutation.mutate(data)}
      />
    </motion.div>
  );
}
