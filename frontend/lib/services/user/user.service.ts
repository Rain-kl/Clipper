import { BaseService } from '../core/base.service';

/**
 * 用户服务
 * 处理用户个人设置相关的 API 请求
 */
export class UserService extends BaseService {
  protected static readonly basePath = '/api/v1/user';
}

