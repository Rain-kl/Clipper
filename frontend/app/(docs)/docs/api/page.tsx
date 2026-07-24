import { getTranslations } from 'next-intl/server';
import { LegalPageLayout } from '@/components/common/docs/legal-page-layout';
import {
  getApiSections,
  DOCS_LAST_UPDATED,
} from '@/components/common/docs/api';

export default async function ApiDocPage() {
  const t = await getTranslations('docs');

  return (
    <LegalPageLayout
      title={t('api.title')}
      lastUpdated={DOCS_LAST_UPDATED}
      sections={getApiSections(t)}
      description={
        <p className='text-muted-foreground text-sm leading-relaxed'>
          {t('api.description')}
        </p>
      }
    />
  );
}
