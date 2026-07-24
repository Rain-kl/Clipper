import { type PolicySection } from './types';
import { CodeBlock } from '@/components/ui/code-block';

type TFn = (key: string) => string;

export function getHowToUseSections(t: TFn): PolicySection[] {
  return [
    {
      value: 'quick-start',
      title: t('howToUse.section1.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <div className='bg-muted/50 border border-border/50 rounded-lg px-3 py-2 mb-6'>
            <p className='text-muted-foreground m-0'>
              {t('howToUse.section1.desc')}
            </p>
          </div>
          <ul className='list-disc pl-5 space-y-2'>
            <li>
              <strong>{t('howToUse.section1.architecture')}</strong>
              {t('howToUse.section1.architectureDesc')}
            </li>
            <li>
              <strong>{t('howToUse.section1.authSystem')}</strong>
              {t('howToUse.section1.authSystemDesc')}
            </li>
            <li>
              <strong>{t('howToUse.section1.accessToken')}</strong>
              {t('howToUse.section1.accessTokenDesc')}
            </li>
            <li>
              <strong>{t('howToUse.section1.observability')}</strong>
              {t('howToUse.section1.observabilityDesc')}
            </li>
          </ul>
        </div>
      ),
    },
    {
      value: 'auth-security',
      title: t('howToUse.section2.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('howToUse.section2.desc')}</p>
          <h3
            id='2-1-login'
            className='text-base md:text-lg font-semibold text-foreground mt-4 mb-2'
          >
            {t('howToUse.section2.passwordAuth')}
          </h3>
          <ul className='list-disc pl-4 md:pl-5 space-y-2'>
            <li>
              <strong>{t('howToUse.section2.selfRegister')}</strong>
              {t('howToUse.section2.selfRegisterDesc')}{' '}
              <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
                bcrypt
              </code>{' '}
              {t('howToUse.section2.bcryptDesc')}
            </li>
            <li>
              <strong>{t('howToUse.section2.switchControl')}</strong>
              {t('howToUse.section2.switchControlDesc')}
            </li>
          </ul>

          <h3
            id='2-2-oidc'
            className='text-base md:text-lg font-semibold text-foreground mt-4 mb-2'
          >
            {t('howToUse.section2.oidc')}
          </h3>
          <p>{t('howToUse.section2.oidcDesc')}</p>
          <ol className='list-decimal pl-4 md:pl-5 space-y-1'>
            <li>{t('howToUse.section2.oidcStep1')}</li>
            <li>{t('howToUse.section2.oidcStep2')}</li>
            <li>{t('howToUse.section2.oidcStep3')}</li>
          </ol>
        </div>
      ),
      children: [
        { value: '2-1-login', title: t('howToUse.section2.passwordAuth') },
        { value: '2-2-oidc', title: t('howToUse.section2.oidc') },
      ],
    },
    {
      value: 'access-token',
      title: t('howToUse.section3.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('howToUse.section3.desc')}</p>
          <h3
            id='3-1-generation'
            className='text-base md:text-lg font-semibold text-foreground mt-4 mb-2'
          >
            {t('howToUse.section3.generation')}
          </h3>
          <ul className='list-disc pl-4 md:pl-5 space-y-2'>
            <li>
              <strong>{t('howToUse.section3.oneTimeDisplay')}</strong>
              {t('howToUse.section3.oneTimeDisplayDesc')}{' '}
              <code className='bg-muted px-1.5 py-0.5 rounded text-xs font-mono'>
                at_xxx
              </code>
              ){t('howToUse.section3.oneTimeDisplayHint')}
            </li>
            <li>
              <strong>{t('howToUse.section3.secureHash')}</strong>
              {t('howToUse.section3.secureHashDesc')}
            </li>
          </ul>

          <h3
            id='3-2-usage'
            className='text-base md:text-lg font-semibold text-foreground mt-4 mb-2'
          >
            {t('howToUse.section3.headerUsage')}
          </h3>
          <p>{t('howToUse.section3.headerUsageDesc')}</p>
          <div className='space-y-2'>
            <p className='font-semibold text-xs text-muted-foreground'>
              {t('howToUse.section3.bearerHeader')}
            </p>
            <CodeBlock
              code={`GET /api/v1/user/self HTTP/1.1
Host: localhost:8000
Authorization: Bearer at_628d022b7a95e26b...`}
              language='http'
            />
          </div>
          <div className='space-y-2'>
            <p className='font-semibold text-xs text-muted-foreground'>
              {t('howToUse.section3.customHeader')}
            </p>
            <CodeBlock
              code={`GET /api/v1/user/self HTTP/1.1
Host: localhost:8000
X-Access-Token: at_628d022b7a95e26b...`}
              language='http'
            />
          </div>
        </div>
      ),
      children: [
        { value: '3-1-generation', title: t('howToUse.section3.generation') },
        { value: '3-2-usage', title: t('howToUse.section3.headerUsage') },
      ],
    },
    {
      value: 'config-system',
      title: t('howToUse.section4.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('howToUse.section4.desc')}</p>
          <ul className='list-disc pl-4 md:pl-5 space-y-2'>
            <li>
              <strong>{t('howToUse.section4.cacheAccel')}</strong>
              {t('howToUse.section4.cacheAccelDesc')}
            </li>
            <li>
              <strong>{t('howToUse.section4.coreConfigs')}</strong>
              <ul className='list-[circle] pl-5 mt-1 space-y-1 text-muted-foreground'>
                <li>
                  <code className='bg-muted px-1 rounded text-xs font-mono'>
                    site_name
                  </code>
                  ：{t('howToUse.section4.siteName')}
                </li>
                <li>
                  <code className='bg-muted px-1 rounded text-xs font-mono'>
                    password_login_enabled
                  </code>
                  ：{t('howToUse.section4.passwordLoginEnabled')}
                </li>
                <li>
                  <code className='bg-muted px-1 rounded text-xs font-mono'>
                    registration_enabled
                  </code>
                  ：{t('howToUse.section4.registrationEnabled')}
                </li>
                <li>
                  <code className='bg-muted px-1 rounded text-xs font-mono'>
                    max_api_keys_per_user
                  </code>
                  ：{t('howToUse.section4.maxApiKeys')}
                </li>
              </ul>
            </li>
          </ul>
        </div>
      ),
    },
    {
      value: 'worker-scheduler',
      title: t('howToUse.section5.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('howToUse.section5.desc')}</p>
          <ul className='list-disc pl-4 md:pl-5 space-y-2'>
            <li>
              <strong>{t('howToUse.section5.scheduler')}</strong>
              {t('howToUse.section5.schedulerDesc')}
            </li>
            <li>
              <strong>{t('howToUse.section5.worker')}</strong>
              {t('howToUse.section5.workerDesc')}
            </li>
          </ul>
        </div>
      ),
    },
    {
      value: 'tracing-metrics',
      title: t('howToUse.section6.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('howToUse.section6.desc')}</p>
          <ul className='list-disc pl-4 md:pl-5 space-y-2'>
            <li>
              <strong>{t('howToUse.section6.tracing')}</strong>
              {t('howToUse.section6.tracingDesc')}
            </li>
            <li>
              <strong>{t('howToUse.section6.structuredLog')}</strong>
              {t('howToUse.section6.structuredLogDesc')}
            </li>
          </ul>
        </div>
      ),
    },
  ];
}
