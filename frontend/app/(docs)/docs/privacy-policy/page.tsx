import { getTranslations } from 'next-intl/server';
import { LegalPageLayout } from '@/components/common/docs/legal-page-layout';
import { getPrivacySections } from '@/components/common/docs/privacy';
import { TERMS_LAST_UPDATED } from '@/components/common/docs/terms';

export default async function PrivacyPolicyPage() {
  const t = await getTranslations('docs');

  return (
    <LegalPageLayout
      title={t('privacy.title')}
      lastUpdated={TERMS_LAST_UPDATED}
      sections={getPrivacySections(t)}
      description={
        <span>
          {t('privacy.description')}
          <br className='hidden md:block' />
          {t('privacy.descriptionRights')}
        </span>
      }
    />
  );
}
