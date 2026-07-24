import { type PolicySection } from './types';

type TFn = (key: string) => string;

export function getPrivacySections(t: TFn): PolicySection[] {
  return [
    {
      value: 'collection-details',
      title: t('privacy.section1.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('privacy.section1.desc')}</p>
          <div className='space-y-3'>
            <div>
              <span className='font-semibold text-foreground'>
                {t('privacy.section1.authInfo')}
              </span>
              <p className='mt-1 text-muted-foreground'>
                {t('privacy.section1.authInfoDesc')}
                <strong>{t('privacy.section1.authInfoNote')}</strong>
              </p>
            </div>
            <div>
              <span className='font-semibold text-foreground'>
                {t('privacy.section1.logInfo')}
              </span>
              <p className='mt-1 text-muted-foreground'>
                {t('privacy.section1.logInfoDesc')}
              </p>
            </div>
          </div>
        </div>
      ),
    },
    {
      value: 'storage-security',
      title: t('privacy.section2.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('privacy.section2.desc')}</p>
          <ul className='list-disc pl-4 md:pl-5 space-y-2'>
            <li>
              <strong>{t('privacy.section2.storageSecurity')}</strong>
              {t('privacy.section2.storageSecurityDesc')}
            </li>
            <li>
              <strong>{t('privacy.section2.encryption')}</strong>
              {t('privacy.section2.encryptionDesc')}
            </li>
            <li>
              <strong>{t('privacy.section2.accessControl')}</strong>
              {t('privacy.section2.accessControlDesc')}
            </li>
          </ul>
        </div>
      ),
    },
    {
      value: 'usage-rules',
      title: t('privacy.section3.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('privacy.section3.desc')}</p>
          <ul className='list-disc pl-4 md:pl-5 space-y-1'>
            <li>
              <strong>{t('privacy.section3.identity')}</strong>
              {t('privacy.section3.identityDesc')}
            </li>
            <li>
              <strong>{t('privacy.section3.business')}</strong>
              {t('privacy.section3.businessDesc')}
            </li>
            <li>
              <strong>{t('privacy.section3.riskControl')}</strong>
              {t('privacy.section3.riskControlDesc')}
            </li>
          </ul>
          <p>
            <strong>{t('privacy.section3.prohibited')}</strong>
            {t('privacy.section3.prohibitedDesc')}
          </p>
        </div>
      ),
    },
    {
      value: 'sharing-disclosure',
      title: t('privacy.section4.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>
            <strong>{t('privacy.section4.sharingPrinciple')}</strong>
            {t('privacy.section4.sharingPrincipleDesc')}
          </p>
          <ul className='list-disc pl-4 md:pl-5 space-y-1 text-muted-foreground'>
            <li>{t('privacy.section4.sharingException1')}</li>
            <li>{t('privacy.section4.sharingException2')}</li>
          </ul>
          <p>
            <strong>{t('privacy.section4.transfer')}</strong>
            {t('privacy.section4.transferDesc')}
          </p>
        </div>
      ),
    },
    {
      value: 'user-rights',
      title: t('privacy.section5.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('privacy.section5.desc')}</p>
          <div className='space-y-3'>
            <div>
              <span className='font-semibold text-foreground'>
                {t('privacy.section5.viewManage')}
              </span>
              <p className='mt-1 text-muted-foreground'>
                {t('privacy.section5.viewManageDesc')}
              </p>
            </div>
            <div>
              <span className='font-semibold text-foreground'>
                {t('privacy.section5.deleteAccount')}
              </span>
              <p className='mt-1 text-muted-foreground'>
                {t('privacy.section5.deleteAccountDesc')}
              </p>
            </div>
          </div>
        </div>
      ),
    },
    {
      value: 'policy-update',
      title: t('privacy.section6.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>{t('privacy.section6.desc')}</p>
          <p>{t('privacy.section6.notification')}</p>
        </div>
      ),
    },
  ];
}
