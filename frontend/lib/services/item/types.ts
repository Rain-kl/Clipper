export type ItemContentType = 'text' | 'image' | 'file';
export type ItemLifecycle = 'pending' | 'active' | 'archived' | 'trash';
export type ItemImportance = 'none' | 'fragment' | 'note' | 'vault';

export interface ItemAttachment {
  id: string;
  item_id: string;
  upload_id: string;
  sort: number;
  // optional denormalized from API:
  file_name?: string;
  mime_type?: string;
  file_size?: number;
}

export interface Item {
  id: string;
  user_id: string;
  content_type: ItemContentType;
  title: string;
  body: string;
  lifecycle: ItemLifecycle;
  importance: ItemImportance;
  source: string;
  archived_at?: string | null;
  trashed_at?: string | null;
  created_at: string;
  updated_at: string;
  attachments?: ItemAttachment[];
}

export interface ListItemsParams {
  page?: number;
  page_size?: number;
  q?: string;
  lifecycle?: ItemLifecycle;
  importance?: ItemImportance;
  content_type?: ItemContentType;
  include_archived?: boolean;
  include_trash?: boolean;
}

export interface ListItemsResult {
  total: number;
  results: Item[];
}

export interface TimelineDay {
  date: string; // YYYY-MM-DD server-suggested; client may regroup
  items: Item[];
  archived_count: number;
  archived_items?: Item[];
}

export interface TimelineResult {
  days: TimelineDay[];
}

export interface CreateItemPayload {
  title?: string;
  body?: string;
  upload_ids?: string[];
}

export interface PatchItemPayload {
  title?: string;
  body?: string;
  lifecycle?: ItemLifecycle;
  importance?: ItemImportance;
}
