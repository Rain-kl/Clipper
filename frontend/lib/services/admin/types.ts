

/**
 * 系统配置信息
 */
export interface SystemConfig {
  /** 配置键 */
  key: string;
  /** 配置值 */
  value: string;
  /** 配置类型：'system' | 'business' */
  type: 'system' | 'business';
  /** 配置描述 */
  description: string;
  /** 创建时间 */
  created_at: string;
  /** 更新时间 */
  updated_at: string;
}

/**
 * 创建系统配置请求参数
 */
export interface CreateSystemConfigRequest {
  /** 配置键（最大64字符） */
  key: string;
  /** 配置值（最大255字符） */
  value: string;
  /** 配置类型：'system' | 'business' */
  type: 'system' | 'business';
  /** 配置描述（最大255字符，可选） */
  description?: string;
}

/**
 * 更新系统配置请求参数
 */
export interface UpdateSystemConfigRequest {
  /** 配置值（最大255字符） */
  value: string;
  /** 配置描述（最大255字符，可选） */
  description?: string;
}



// ==================== 任务管理 ====================

/**
 * 任务类型响应
 */
export interface TaskTypeResponse {
  Type?: string;
  type?: string;
  AsynqTask?: string;
  asynq_task?: string;
  Name?: string;
  name?: string;
  Description?: string;
  description?: string;
  SupportsTime?: boolean;
  supports_time?: boolean;
  MaxRetry?: number;
  max_retry?: number;
  Queue?: string;
  queue?: string;
}

/**
 * 任务元数据
 */
export interface TaskMeta {
  /** 任务类型标识 */
  type: string;
  /** Asynq 任务名称 */
  asynq_task: string;
  /** 任务名称 */
  name: string;
  /** 任务描述 */
  description: string;
  /** 是否支持时间范围参数 */
  supports_time: boolean;
  /** 最大重试次数 */
  max_retry: number;
  /** 队列名称 */
  queue: string;
}

/**
 * 下发任务请求参数
 */
export interface DispatchTaskRequest {
  /** 任务类型 */
  task_type: string;
  /** 开始时间（可选，仅部分任务支持） */
  start_time?: string;
  /** 结束时间（可选，仅部分任务支持） */
  end_time?: string;
  /** 用户 ID（可选，仅部分任务需要） */
  user_id?: string;
}

export type TaskExecutionStatus = 'pending' | 'running' | 'succeeded' | 'failed';

/**
 * 任务执行记录
 */
export interface TaskExecution {
  id: string;
  task_id: string;
  task_type: string;
  task_name: string;
  status: TaskExecutionStatus;
  retryable: boolean;
  max_retry: number;
  retry_count: number;
  log: string;
  error_message: string;
  result: string;
  started_at?: string | null;
  finished_at?: string | null;
  duration: number;
  payload: string;
  triggered_by: string;
  created_at: string;
  updated_at: string;
}

/**
 * 查询任务执行记录请求参数
 */
export interface ListTaskExecutionsRequest {
  status?: TaskExecutionStatus;
  task_type?: string;
  page?: number;
  page_size?: number;
}

/**
 * 查询任务执行记录响应
 */
export interface ListTaskExecutionsResponse {
  items: TaskExecution[];
  total: number;
  page: number;
  page_size: number;
}

// ==================== 用户管理 ====================

/**
 * 管理员用户信息
 */
export interface AdminUser {
  /** 用户 ID */
  id: string;
  /** 用户名 */
  username: string;
  /** 昵称 */
  nickname: string;
  /** 头像 URL */
  avatar_url: string;
  /** 信任等级 */
  trust_level: number;
  /** 支付积分 */
  pay_score: number;
  /** 累计收入 */
  total_receive: string;
  /** 累计支出 */
  total_payment: string;
  /** 累计社区积分 */
  total_community: string;
  /** 社区余额 */
  community_balance: string;
  /** 可用余额 */
  available_balance: string;
  /** 是否激活 */
  is_active: boolean;
  /** 是否管理员 */
  is_admin: boolean;
  /** 最后登录时间 */
  last_login_at: string;
  /** 创建时间 */
  created_at: string;
  /** 更新时间 */
  updated_at: string;
}

/**
 * 用户列表查询请求参数
 */
export interface ListUsersRequest {
  /** 页码（从 1 开始） */
  page: number;
  /** 每页数量（1-100） */
  page_size: number;
  /** 用户 ID 精确过滤（可选） */
  user_id?: string;
  /** 用户名前缀过滤（可选） */
  username?: string;
}

/**
 * 用户列表响应
 */
export interface ListUsersResponse {
  /** 用户列表 */
  users: AdminUser[];
  /** 总数 */
  total: number;
}

/**
 * 更新用户状态请求参数
 */
export interface UpdateUserStatusRequest {
  /** 是否激活 */
  is_active: boolean;
}

/**
 * 认证源信息
 */
export interface AuthSource {
  id: string;
  name: string;
  type: 'oidc';
  display_name: string;
  is_active: boolean;
  client_id: string;
  client_secret?: string;
  client_secret_configured?: boolean;
  openid_discovery_url: string;
  scopes: string;
  icon_url: string;
  created_at: string;
  updated_at: string;
}

export interface AuthSourceRequest {
  name: string;
  type: 'oidc';
  display_name: string;
  is_active: boolean;
  client_id: string;
  client_secret: string;
  openid_discovery_url: string;
  scopes: string;
  icon_url: string;
}

export interface ToggleAuthSourceRequest {
  is_active: boolean;
}
