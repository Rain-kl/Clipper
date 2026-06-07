/**
 * 公共配置响应
 */
export interface PublicConfigResponse {
  /** 允许上传的图片扩展名 */
  upload_allowed_extensions: string;
  /** 站点名称 */
  site_name: string;
  /** 是否允许注册 */
  registration_enabled: boolean;
  /** 每个用户最大 API Key 数量 */
  max_api_keys_per_user: number;
}
