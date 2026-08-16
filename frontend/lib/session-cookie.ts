/** Must match backend `app.session_cookie_name` (config.example.yaml). */
export const DEFAULT_SESSION_COOKIE_NAME = 'clipper_session_id';

/** Resolve the session cookie name used by Next.js `proxy.ts`. */
export function resolveSessionCookieName(
  env: NodeJS.ProcessEnv = process.env,
): string {
  return (
    env.WAVELET_SESSION_COOKIE_NAME ||
    env.CLIPPER_SESSION_COOKIE_NAME ||
    DEFAULT_SESSION_COOKIE_NAME
  );
}
