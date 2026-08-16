// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

import { BaseService } from '@/lib/services/core';
import type {
  BindMessageChannelRequest,
  MessageBinding,
  PublicMessageChannel,
} from './types';

export class UserMessageGatewayService extends BaseService {
  protected static readonly basePath = '/api/v1/message-gateway';

  static async listBindings(): Promise<MessageBinding[]> {
    return this.get<MessageBinding[]>('/bindings');
  }

  static async listChannels(): Promise<PublicMessageChannel[]> {
    return this.get<PublicMessageChannel[]>('/channels');
  }

  static async bind(data: BindMessageChannelRequest): Promise<MessageBinding> {
    return this.post<MessageBinding>('/bindings', data);
  }

  static async unbind(id: string): Promise<void> {
    return this.delete<void>(`/bindings/${id}`);
  }
}
