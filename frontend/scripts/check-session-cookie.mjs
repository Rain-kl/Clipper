import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const cookieHelper = readFileSync(
  resolve(root, 'lib/session-cookie.ts'),
  'utf8',
);
const proxy = readFileSync(resolve(root, 'proxy.ts'), 'utf8');
const envExample = readFileSync(resolve(root, '.env.example'), 'utf8');

const errors = [];

if (!/['"]clipper_session_id['"]/.test(cookieHelper)) {
  errors.push(
    'frontend/lib/session-cookie.ts must default the session cookie to clipper_session_id (backend app.session_cookie_name)',
  );
}

if (!proxy.includes('resolveSessionCookieName')) {
  errors.push('frontend/proxy.ts must use resolveSessionCookieName()');
}

if (/WAVELET_SESSION_COOKIE_NAME=wavelet_session_id/.test(envExample)) {
  errors.push(
    'frontend/.env.example still defaults WAVELET_SESSION_COOKIE_NAME to wavelet_session_id',
  );
}

if (!/WAVELET_SESSION_COOKIE_NAME=clipper_session_id/.test(envExample)) {
  errors.push(
    'frontend/.env.example must set WAVELET_SESSION_COOKIE_NAME=clipper_session_id',
  );
}

if (errors.length > 0) {
  console.error(errors.join('\n'));
  process.exit(1);
}

console.log('session cookie config OK');
