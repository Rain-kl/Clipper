/**
 * Clipper Item 服务模块
 *
 * @description
 * 提供剪藏条目 CRUD、时间线与统计相关 API
 */

export { ItemService } from './item.service';
export type {
  ItemContentType,
  ItemLifecycle,
  ItemImportance,
  ItemAttachment,
  Item,
  ListItemsParams,
  ListItemsResult,
  TimelineDay,
  TimelineResult,
  CreateItemPayload,
  PatchItemPayload,
} from './types';
