import { type PolicySection } from './types';
import { CodeBlock } from '@/components/ui/code-block';
import {
  DocsTable,
  DocsTableBody,
  DocsTableCell,
  DocsTableHead,
  DocsTableHeader,
  DocsTableRow,
} from '@/components/ui/docs-table';

export const DOCS_LAST_UPDATED = '2026-06-07';

type TFn = (key: string) => string;

export function getApiSections(t: TFn): PolicySection[] {
  return [
    {
      value: 'api-specs',
      title: t('api.section1.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <div className='bg-muted/50 border border-border/50 rounded-lg px-3 py-2 mb-6'>
            <p className='text-muted-foreground m-0'>
              {t('api.section1.desc')}
            </p>
          </div>

          <h3
            id='1-1-response-format'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-3'
          >
            {t('api.section1.responseFormat')}
          </h3>
          <p>{t('api.section1.responseFormatDesc')}</p>
          <DocsTable>
            <DocsTableHeader>
              <DocsTableRow>
                <DocsTableHead className='w-[120px]'>
                  {t('api.field')}
                </DocsTableHead>
                <DocsTableHead className='w-[100px]'>
                  {t('api.type')}
                </DocsTableHead>
                <DocsTableHead>{t('api.description')}</DocsTableHead>
              </DocsTableRow>
            </DocsTableHeader>
            <DocsTableBody>
              <DocsTableRow>
                <DocsTableCell className='font-mono text-xs'>
                  error_msg
                </DocsTableCell>
                <DocsTableCell>string</DocsTableCell>
                <DocsTableCell>{t('api.section1.errorMsgDesc')}</DocsTableCell>
              </DocsTableRow>
              <DocsTableRow>
                <DocsTableCell className='font-mono text-xs'>
                  data
                </DocsTableCell>
                <DocsTableCell>any</DocsTableCell>
                <DocsTableCell>{t('api.section1.dataDesc')}</DocsTableCell>
              </DocsTableRow>
            </DocsTableBody>
          </DocsTable>

          <p className='mt-2'>{t('api.section1.successExample')}</p>
          <CodeBlock
            code={`{
  "error_msg": "",
  "data": {
    "id": 1,
    "username": "ryan",
    "nickname": "Ryan"
  }
}`}
            language='json'
          />

          <p className='mt-2'>{t('api.section1.failExample')}</p>
          <CodeBlock
            code={`{
  "error_msg": "${t('api.section1.failExampleErrorMsg')}",
  "data": null
}`}
            language='json'
          />

          <h3
            id='1-2-authentication'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-3'
          >
            {t('api.section1.authentication')}
          </h3>
          <p>{t('api.section1.authenticationDesc')}</p>
          <ul className='list-disc pl-5 space-y-2'>
            <li>
              <strong>{t('api.section1.sessionCredential')}</strong>
              {t('api.section1.sessionCredentialDesc')}
            </li>
            <li>
              <strong>{t('api.section1.accessTokenCredential')}</strong>
              {t('api.section1.accessTokenCredentialDesc')}
            </li>
          </ul>
          <div className='bg-muted border rounded-xl p-4 mt-2 space-y-2'>
            <p className='font-bold text-xs'>
              {t('api.section1.supportedHeaders')}
            </p>
            <ul className='list-disc pl-5 text-xs text-muted-foreground space-y-1'>
              <li>
                <code className='bg-muted-foreground/10 px-1 rounded text-[11px] font-mono'>
                  Authorization: Bearer at_xxx
                </code>
              </li>
              <li>
                <code className='bg-muted-foreground/10 px-1 rounded text-[11px] font-mono'>
                  X-Access-Token: at_xxx
                </code>
              </li>
            </ul>
          </div>
        </div>
      ),
      children: [
        {
          value: '1-1-response-format',
          title: t('api.section1.responseFormat'),
        },
        {
          value: '1-2-authentication',
          title: t('api.section1.authentication'),
        },
      ],
    },
    {
      value: 'auth-apis',
      title: t('api.section2.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <h3
            id='2-1-register'
            className='text-base md:text-lg font-semibold text-foreground mt-4 mb-2'
          >
            {t('api.section2.register')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>POST{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/user/register
            </code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section2.registerDesc')}
          </p>
          <DocsTable>
            <DocsTableHeader>
              <DocsTableRow>
                <DocsTableHead className='w-[120px]'>
                  {t('api.param')}
                </DocsTableHead>
                <DocsTableHead className='w-[80px]'>
                  {t('api.required')}
                </DocsTableHead>
                <DocsTableHead>{t('api.type')}</DocsTableHead>
                <DocsTableHead>{t('api.description')}</DocsTableHead>
              </DocsTableRow>
            </DocsTableHeader>
            <DocsTableBody>
              <DocsTableRow>
                <DocsTableCell className='font-mono text-xs'>
                  username
                </DocsTableCell>
                <DocsTableCell>{t('api.yes')}</DocsTableCell>
                <DocsTableCell>string</DocsTableCell>
                <DocsTableCell>{t('api.section2.usernameDesc')}</DocsTableCell>
              </DocsTableRow>
              <DocsTableRow>
                <DocsTableCell className='font-mono text-xs'>
                  password
                </DocsTableCell>
                <DocsTableCell>{t('api.yes')}</DocsTableCell>
                <DocsTableCell>string</DocsTableCell>
                <DocsTableCell>{t('api.section2.passwordDesc')}</DocsTableCell>
              </DocsTableRow>
              <DocsTableRow>
                <DocsTableCell className='font-mono text-xs'>
                  nickname
                </DocsTableCell>
                <DocsTableCell>{t('api.no')}</DocsTableCell>
                <DocsTableCell>string</DocsTableCell>
                <DocsTableCell>{t('api.section2.nicknameDesc')}</DocsTableCell>
              </DocsTableRow>
            </DocsTableBody>
          </DocsTable>

          <h3
            id='2-2-login'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-2'
          >
            {t('api.section2.passwordLogin')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>POST{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/user/login
            </code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section2.passwordLoginDesc')}
          </p>
          <DocsTable>
            <DocsTableHeader>
              <DocsTableRow>
                <DocsTableHead className='w-[120px]'>
                  {t('api.param')}
                </DocsTableHead>
                <DocsTableHead className='w-[80px]'>
                  {t('api.required')}
                </DocsTableHead>
                <DocsTableHead>{t('api.type')}</DocsTableHead>
                <DocsTableHead>{t('api.description')}</DocsTableHead>
              </DocsTableRow>
            </DocsTableHeader>
            <DocsTableBody>
              <DocsTableRow>
                <DocsTableCell className='font-mono text-xs'>
                  username
                </DocsTableCell>
                <DocsTableCell>{t('api.yes')}</DocsTableCell>
                <DocsTableCell>string</DocsTableCell>
                <DocsTableCell>{t('api.username')}</DocsTableCell>
              </DocsTableRow>
              <DocsTableRow>
                <DocsTableCell className='font-mono text-xs'>
                  password
                </DocsTableCell>
                <DocsTableCell>{t('api.yes')}</DocsTableCell>
                <DocsTableCell>string</DocsTableCell>
                <DocsTableCell>{t('api.password')}</DocsTableCell>
              </DocsTableRow>
            </DocsTableBody>
          </DocsTable>

          <h3
            id='2-3-logout'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-2'
          >
            {t('api.section2.logout')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>GET{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/user/logout
            </code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section2.logoutDesc')}
          </p>

          <h3
            id='2-4-profile'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-2'
          >
            {t('api.section2.getProfile')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>GET{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/user/self
            </code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section2.getProfileDesc')}
          </p>
        </div>
      ),
      children: [
        { value: '2-1-register', title: t('api.section2.register') },
        { value: '2-2-login', title: t('api.section2.passwordLogin') },
        { value: '2-3-logout', title: t('api.section2.logout') },
        { value: '2-4-profile', title: t('api.section2.getProfile') },
      ],
    },
    {
      value: 'token-apis',
      title: t('api.section3.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <div className='bg-muted/50 border border-border/50 rounded-lg px-3 py-2 mb-4'>
            <p className='text-muted-foreground m-0'>
              {t('api.section3.desc')}
            </p>
          </div>

          <h3
            id='3-1-list-token'
            className='text-base md:text-lg font-semibold text-foreground mt-4 mb-2'
          >
            {t('api.section3.listTokens')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>GET{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/user/access-tokens
            </code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section3.listTokensDesc')}
          </p>

          <h3
            id='3-2-create-token'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-2'
          >
            {t('api.section3.createToken')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>POST{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/user/access-tokens
            </code>
          </p>
          <p>
            <strong>{t('api.param')}</strong>JSON Body{' '}
            <code className='bg-muted px-1.5 rounded text-xs font-mono'>{`{"name": "${t('api.section3.tokenName')}", "is_admin": false}`}</code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section3.createTokenDesc')}
          </p>
          <p className='mt-1 text-xs text-muted-foreground'>
            <code className='bg-muted px-1 rounded'>is_admin</code>（
            {t('api.optional')}，{t('api.default')}{' '}
            <code className='bg-muted px-1 rounded'>false</code>
            ）：{t('api.section3.isAdminDesc')}{' '}
            <code className='bg-muted px-1 rounded'>/admin/**</code>{' '}
            {t('api.section3.endpoint')}.
          </p>
          <p className='mt-2'>{t('api.section3.successResponseExample')}</p>
          <CodeBlock
            code={`{
  "error_msg": "",
  "data": {
    "token": "at_628d022b7a95e26bcd8b29c9...",
    "record": {
      "id": 5,
      "user_id": 1,
      "name": "my-dev-key",
      "masked_token": "at_628d...29c9",
      "is_admin": false,
      "created_at": "2026-06-07T21:30:00+08:00"
    }
  }
}`}
            language='json'
          />

          <h3
            id='3-3-delete-token'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-2'
          >
            {t('api.section3.deleteToken')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>DELETE{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/user/access-tokens/:id
            </code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section3.deleteTokenDesc')}
          </p>

          <h3
            id='3-4-rotate-token'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-2'
          >
            {t('api.section3.rotateToken')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>POST{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/user/access-tokens/:id/rotate
            </code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section3.rotateTokenDesc')}
          </p>
        </div>
      ),
      children: [
        { value: '3-1-list-token', title: t('api.section3.listTokens') },
        { value: '3-2-create-token', title: t('api.section3.createToken') },
        { value: '3-3-delete-token', title: t('api.section3.deleteToken') },
        { value: '3-4-rotate-token', title: t('api.section3.rotateToken') },
      ],
    },
    {
      value: 'config-apis',
      title: t('api.section4.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <h3
            id='4-1-public-config'
            className='text-base md:text-lg font-semibold text-foreground mt-4 mb-2'
          >
            {t('api.section4.publicConfig')}
          </h3>
          <p>
            <strong>{t('api.endpoint')}</strong>GET{' '}
            <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
              /api/v1/config/public
            </code>
          </p>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section4.publicConfigDesc')}
          </p>
          <p className='mt-2'>{t('api.section4.responseExample')}</p>
          <CodeBlock
            code={`{
  "error_msg": "",
  "data": {
    "site_name": "Wavelet",
    "registration_enabled": "true",
    "password_login_enabled": "true",
    "password_register_enabled": "true",
    "oidc_login_enabled": "true"
  }
}`}
            language='json'
          />

          <h3
            id='4-2-admin-configs'
            className='text-base md:text-lg font-semibold text-foreground mt-6 mb-2'
          >
            {t('api.section4.adminConfigs')}
          </h3>
          <p>
            <strong>{t('api.desc')}</strong>
            {t('api.section4.adminConfigsDesc')}
          </p>
          <ul className='list-disc pl-5 space-y-2'>
            <li>
              <strong>{t('api.section4.getConfigList')}</strong>GET{' '}
              <code className='bg-muted px-1 rounded text-xs font-mono'>
                /api/v1/admin/system-configs?type=system
              </code>
            </li>
            <li>
              <strong>{t('api.section4.createConfig')}</strong>POST{' '}
              <code className='bg-muted px-1 rounded text-xs font-mono'>
                /api/v1/admin/system-configs
              </code>
            </li>
            <li>
              <strong>{t('api.section4.updateConfig')}</strong>PUT{' '}
              <code className='bg-muted px-1 rounded text-xs font-mono'>
                /api/v1/admin/system-configs/:key
              </code>
            </li>
            <li>
              <strong>{t('api.section4.deleteConfig')}</strong>DELETE{' '}
              <code className='bg-muted px-1 rounded text-xs font-mono'>
                /api/v1/admin/system-configs/:key
              </code>
            </li>
          </ul>
        </div>
      ),
      children: [
        { value: '4-1-public-config', title: t('api.section4.publicConfig') },
        { value: '4-2-admin-configs', title: t('api.section4.adminConfigs') },
      ],
    },
  ];
}
