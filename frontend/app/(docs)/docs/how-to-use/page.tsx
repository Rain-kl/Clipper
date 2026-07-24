import { getTranslations } from 'next-intl/server';
import { LegalPageLayout } from '@/components/common/docs/legal-page-layout';
import { getHowToUseSections } from '@/components/common/docs/how-to-use';
import { DOCS_LAST_UPDATED } from '@/components/common/docs/api';

export default async function HowToUsePage() {
  const t = await getTranslations('docs');

  return (
    <LegalPageLayout
      title={t('howToUse.title')}
      lastUpdated={DOCS_LAST_UPDATED}
      sections={getHowToUseSections(t)}
      description={
        <p className='text-muted-foreground text-sm leading-relaxed'>
          {t('howToUse.description')}
        </p>
      }
    />
  );
}
