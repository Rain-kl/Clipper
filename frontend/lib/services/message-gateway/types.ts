// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

export type MessageChannelType = 'telegram' | 'qq';

export interface MessageChannelField {
  key: string;
  type?: string;
  required?: boolean;
}

export interface MessageChannelDefinition {
  type: MessageChannelType | string;
  name: string;
  fields: MessageChannelField[];
}

export interface MessageChannel {
  id: string;
  name: string;
  type: MessageChannelType | string;
  owner_scope: string;
  enabled: boolean;
  bot_token?: string;
  app_id?: string;
  app_secret?: string;
  base_url?: string;
  portal_host?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateMessageChannelRequest {
  name: string;
  type: string;
  enabled?: boolean;
  bot_token?: string;
  app_id?: string;
  app_secret?: string;
  base_url?: string;
  portal_host?: string;
}

export interface UpdateMessageChannelRequest {
  name?: string;
  enabled?: boolean;
  bot_token?: string;
  app_id?: string;
  app_secret?: string;
  base_url?: string;
  portal_host?: string;
}

export interface PublicMessageChannel {
  id: string;
  name: string;
  type: MessageChannelType | string;
}

export interface MessageBinding {
  id: string;
  user_id: string;
  channel_id: string;
  channel_name: string;
  channel_type: string;
  platform_user_id: string;
  created_at: string;
}

export interface BindMessageChannelRequest {
  channel_id: string;
  code: string;
}
