import type {
  ItemContentType,
  ItemImportance,
  ItemLifecycle,
} from '@/lib/services/item/types';

export const LIFECYCLE_OPTIONS: ItemLifecycle[] = [
  'pending',
  'active',
  'archived',
  'trash',
];

export const IMPORTANCE_OPTIONS: ItemImportance[] = [
  'none',
  'fragment',
  'note',
  'vault',
];

export const CONTENT_TYPE_OPTIONS: ItemContentType[] = [
  'text',
  'image',
  'file',
];
