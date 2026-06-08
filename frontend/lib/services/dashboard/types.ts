/**
 * Dashboard 服务类型定义
 */

/**
 * 每日统计项
 */
export interface DailyStatsItem {
  /** 日期，格式: YYYY-MM-DD */
  date: string;
  /** 当日收入金额 */
  income: string;
  /** 当日支出金额 */
  expense: string;
}

/**
 * 每日统计响应
 */
export type DailyStatsResponse = DailyStatsItem[];

/**
 * 获取每日统计请求参数
 */
export interface GetDailyStatsRequest {
  /** 查询天数，最大7天，最小1天 */
  days: number;
}

/**
 * Top 客户项
 */
export interface TopCustomer {
  /** 客户用户ID */
  user_id: number;
  /** 客户用户名 */
  username: string;
  /** 累计付款金额 */
  total_amount: string;
  /** 订单数量 */
  order_count: number;
}

/**
 * Top 客户响应
 */
export type TopCustomersResponse = TopCustomer[];

/**
 * 获取 Top 客户请求参数
 */
export interface GetTopCustomersRequest {
  /** 查询天数，最大7天，最小1天 */
  days: number;
  /** 返回数量，最大10条，最小1条 */
  limit: number;
}
