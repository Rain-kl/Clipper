/**
 * 公共配置响应
 */
export interface PublicConfigResponse {
  /** 允许上传的图片扩展名 */
  upload_allowed_extensions: string;
  /** 站点名称 */
  site_name: string;
  /** 是否允许密码登录 */
  password_login_enabled: boolean;
  /** 是否允许注册 */
  registration_enabled: boolean;
  /** 是否允许密码注册 */
  password_register_enabled: boolean;
  /** 是否允许 OIDC 登录 */
  oidc_login_enabled: boolean;
  /** 每个用户最大 API Key 数量 */
  max_api_keys_per_user: number;
  /** 是否启用 Cap 人机验证 */
  cap_login_enabled: boolean;
  /** 是否自动解题 */
  cap_auto_solve: boolean;
}
