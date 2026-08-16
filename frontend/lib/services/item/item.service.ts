import { BaseService } from '../core/base.service';
import type {
  CreateItemPayload,
  Item,
  ListItemsParams,
  ListItemsResult,
  PatchItemPayload,
  TimelineResult,
} from './types';

export class ItemService extends BaseService {
  protected static readonly basePath = '/api/v1/items';

  static create(payload: CreateItemPayload) {
    return this.post<Item>('', payload);
  }

  static list(params: ListItemsParams = {}) {
    return this.get<ListItemsResult>('', params as Record<string, unknown>);
  }

  static getById(id: string) {
    return this.get<Item>(`/${id}`);
  }

  /** Partial update (HTTP PATCH). Named `update` to avoid shadowing BaseService.patch. */
  static update(id: string, payload: PatchItemPayload) {
    return this.patch<Item>(`/${id}`, payload);
  }

  static remove(id: string, force = false) {
    const q = force ? '?force=1' : '';
    return this.delete<void>(`/${id}${q}`);
  }

  static timeline(params?: {
    expand_archived?: boolean;
    before?: string;
    limit?: number;
  }) {
    return this.get<TimelineResult>(
      '/timeline',
      params as Record<string, unknown>,
    );
  }

  static stats() {
    return this.get<Record<string, number>>('/stats');
  }
}
