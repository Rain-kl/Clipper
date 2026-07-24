import { type PolicySection } from './types';

export const TERMS_LAST_UPDATED = '2026-06-07';

type TFn = (key: string) => string;

export function getTermsSections(t: TFn): PolicySection[] {
  return [
    {
      value: 'contract-establishment',
      title: t('terms.section1.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>
            <strong>{t('terms.section1.subject')}</strong>
            {t('terms.section1.subjectDesc')}
          </p>
          <p>
            <strong>{t('terms.section1.readCarefully')}</strong>
            {t('terms.section1.readCarefullyDesc')}
          </p>
          <p>
            <strong>{t('terms.section1.composition')}</strong>
            {t('terms.section1.compositionDesc')}
          </p>
        </div>
      ),
    },
    {
      value: 'service-definition',
      title: t('terms.section2.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <ul className='list-disc pl-4 md:pl-5 space-y-2'>
            <li>
              <strong>{t('terms.section2.liability')}</strong>
              {t('terms.section2.liabilityDesc')}
            </li>
          </ul>
        </div>
      ),
    },
    {
      value: 'account-specifications',
      title: t('terms.section3.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>
            <strong>{t('terms.section3.accountSystem')}</strong>
            {t('terms.section3.accountSystemDesc')}
          </p>
          <p>
            <strong>{t('terms.section3.securityResponsibility')}</strong>
          </p>
          <ul className='list-disc pl-4 md:pl-5 space-y-2'>
            <li>
              <strong>{t('terms.section3.passwordSecurity')}</strong>
              {t('terms.section3.passwordSecurityDesc')}
            </li>
            <li>
              <strong>{t('terms.section3.tokenSecurity')}</strong>
              {t('terms.section3.tokenSecurityDesc')}
              <strong>{t('terms.section3.tokenLeakLiability')}</strong>
            </li>
          </ul>
        </div>
      ),
    },
    {
      value: 'user-conduct',
      title: t('terms.section4.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>
            {t('terms.section4.desc')}
            <strong>{t('terms.section4.redLine')}</strong>
          </p>
          <div className='bg-red-500/5 border border-red-500/20 rounded-lg p-3 md:p-4 space-y-3'>
            <ul className='list-disc pl-4 md:pl-5 space-y-2 text-red-600 dark:text-red-400 font-medium'>
              <li>
                <strong>{t('terms.section4.nationalSecurity')}</strong>
                {t('terms.section4.nationalSecurityDesc')}
              </li>
              <li>
                <strong>{t('terms.section4.illegalInfo')}</strong>
                {t('terms.section4.illegalInfoDesc')}
              </li>
              <li>
                <strong>{t('terms.section4.harmfulContent')}</strong>
                {t('terms.section4.harmfulContentDesc')}
              </li>
              <li>
                <strong>{t('terms.section4.ipInfringement')}</strong>
                {t('terms.section4.ipInfringementDesc')}
              </li>
              <li>
                <strong>{t('terms.section4.otherIllegal')}</strong>
                {t('terms.section4.otherIllegalDesc')}
              </li>
            </ul>
          </div>
          <p>
            <strong>{t('terms.section4.enforcement')}</strong>
            {t('terms.section4.enforcementDesc')}
            <strong>{t('terms.section4.enforcementAction')}</strong>
          </p>
        </div>
      ),
    },
    {
      value: 'liability-limitation',
      title: t('terms.section5.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>
            <strong>{t('terms.section5.basicDisclaimer')}</strong>
            {t('terms.section5.basicDisclaimerDesc')}
          </p>
          <p>
            <strong>{t('terms.section5.techInterruption')}</strong>
            {t('terms.section5.techInterruptionDesc')}
          </p>
          <ul className='list-disc pl-4 md:pl-5 space-y-1 text-muted-foreground'>
            <li>{t('terms.section5.naturalDisaster')}</li>
            <li>{t('terms.section5.govAction')}</li>
            <li>{t('terms.section5.telecomFailure')}</li>
            <li>{t('terms.section5.hacking')}</li>
          </ul>
        </div>
      ),
    },
    {
      value: 'governing-law',
      title: t('terms.section6.title'),
      content: (
        <div className='space-y-4 text-sm leading-relaxed'>
          <p>
            <strong>{t('terms.section6.applicableLaw')}</strong>
            {t('terms.section6.applicableLawDesc')}
            <strong>{t('terms.section6.lawName')}</strong>
            {t('terms.section6.lawScope')}
          </p>
          <p>
            <strong>{t('terms.section6.disputeResolution')}</strong>
            {t('terms.section6.disputeResolutionDesc')}
          </p>
        </div>
      ),
    },
  ];
}
