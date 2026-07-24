/**
 * API 配置
 *
 * 浏览器默认走同源相对路径（`/api/...`），由 `next.config` rewrites 反代到后端，
 * 避免 dev 下跨域。仅当设置了 `NEXT_PUBLIC_WAVELET_BACKEND_URL` 时浏览器才直连后端
 *（静态导出 embed / WebSocket 等场景）。
 */

/**
 * 获取 API 基础 URL
 * @returns API 基础 URL；浏览器侧空字符串表示同源相对路径
 */
export function getApiBaseUrl(): string {
  if (typeof window === 'undefined') {
    return (
      process.env.WAVELET_BACKEND_URL ||
      process.env.NEXT_PUBLIC_WAVELET_BACKEND_URL ||
      'http://localhost:8000'
    );
  }
  // 空字符串 → axios 请求 `/api/...`，经 Next rewrites 到后端，无 CORS 问题
  return process.env.NEXT_PUBLIC_WAVELET_BACKEND_URL || '';
}

/**
 * API 配置选项
 */
export const apiConfig = {
  /** Basic URL */
  baseURL: getApiBaseUrl(),
  /** 超时时间（毫秒） */
  timeout: 15000,
  /** 携带凭证 */
  withCredentials: true,
} as const;
