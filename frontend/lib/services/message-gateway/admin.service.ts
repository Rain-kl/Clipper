// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

import { BaseService } from '@/lib/services/core';
import type {
  CreateMessageChannelRequest,
  MessageChannel,
  MessageChannelDefinition,
  UpdateMessageChannelRequest,
} from './types';

export class AdminMessageGatewayService extends BaseService {
  protected static readonly basePath = '/api/v1/admin/message-gateway';

  static async list(): Promise<MessageChannel[]> {
    return this.get<MessageChannel[]>('/channels');
  }

  static async definitions(): Promise<MessageChannelDefinition[]> {
    return this.get<MessageChannelDefinition[]>('/channels/definitions');
  }

  static async create(
    data: CreateMessageChannelRequest,
  ): Promise<MessageChannel> {
    return this.post<MessageChannel>('/channels', data);
  }

  static async update(
    id: string,
    data: UpdateMessageChannelRequest,
  ): Promise<MessageChannel> {
    return this.patch<MessageChannel>(`/channels/${id}`, data);
  }

  static async remove(id: string): Promise<void> {
    return this.delete<void>(`/channels/${id}`);
  }

  static async test(id: string): Promise<void> {
    return this.post<void>(`/channels/${id}/test`);
  }
}
