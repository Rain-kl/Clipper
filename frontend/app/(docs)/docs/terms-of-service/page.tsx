import { getTranslations } from 'next-intl/server';
import { LegalPageLayout } from '@/components/common/docs/legal-page-layout';
import {
  TERMS_LAST_UPDATED,
  getTermsSections,
} from '@/components/common/docs/terms';

export default async function TermsOfServicePage() {
  const t = await getTranslations('docs');

  return (
    <LegalPageLayout
      title={t('terms.title')}
      lastUpdated={TERMS_LAST_UPDATED}
      sections={getTermsSections(t)}
      description={<span>{t('terms.description')}</span>}
    />
  );
}
