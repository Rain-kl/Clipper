import * as React from 'react';
import { cn } from '@/lib/utils';
import Link from 'next/link';
import { Github, LucideIcon, WavesIcon } from 'lucide-react';
import { useTranslations } from 'next-intl';

export interface FooterSectionProps {
  className?: string;
}

export const FooterSection = React.memo(function FooterSection({
  className,
}: FooterSectionProps) {
  const t = useTranslations('home.footer');

  return (
    <footer
      className={cn(
        'relative z-10 w-full bg-transparent border-t border-white/10 mt-0 backdrop-blur-sm',
        className,
      )}
    >
      <div className='container mx-auto max-w-7xl px-6 py-20 lg:py-32'>
        <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-12 lg:gap-8 mb-20'>
          <div className='lg:col-span-2 space-y-6'>
            <Link href='/' className='flex items-center gap-2'>
              <span className='flex size-9 items-center justify-center rounded-full bg-foreground text-background [@media(max-height:700px)]:size-8'>
                <WavesIcon aria-hidden='true' />
              </span>
              <span className='text-2xl font-bold tracking-tight text-foreground'>
                Wavelet
              </span>
            </Link>
            <p className='text-muted-foreground text-base leading-relaxed max-w-sm'>
              {t('description')}
            </p>
            <div className='flex gap-4 pt-2'>
              <SocialLink
                icon={Github}
                href='https://github.com/Rain-kl/Wavelet/'
              />
            </div>
          </div>

          <div className='lg:col-span-1'>
            <h3 className='font-semibold text-foreground mb-6'>
              {t('product')}
            </h3>
            <ul className='space-y-4 text-sm text-muted-foreground'>
              <li>
                <FooterLink href='/home'>{t('dashboard')}</FooterLink>
              </li>
              <li>
                <FooterLink href='/settings'>
                  {t('personalSettings')}
                </FooterLink>
              </li>
            </ul>
          </div>

          <div className='lg:col-span-1'>
            <h3 className='font-semibold text-foreground mb-6'>
              {t('development')}
            </h3>
            <ul className='space-y-4 text-sm text-muted-foreground'>
              <li>
                <FooterLink href='/docs/how-to-use'>
                  {t('quickStart')}
                </FooterLink>
              </li>
              <li>
                <FooterLink href='/docs/api'>{t('apiDocs')}</FooterLink>
              </li>
              <li>
                <FooterLink href='https://github.com/Rain-kl/Wavelet'>
                  {t('sourceCode')}
                </FooterLink>
              </li>
            </ul>
          </div>

          <div className='lg:col-span-1'>
            <h3 className='font-semibold text-foreground mb-6'>
              {t('community')}
            </h3>
            <ul className='space-y-4 text-sm text-muted-foreground'>
              <li>
                <FooterLink href='https://github.com/Rain-kl/Wavelet/issues'>
                  GitHub Issues
                </FooterLink>
              </li>
              <li>
                <FooterLink href='https://github.com/Rain-kl/Wavelet/discussions'>
                  {t('discussions')}
                </FooterLink>
              </li>
            </ul>
          </div>
        </div>

        <div className='pt-8 border-t border-border flex flex-col md:flex-row justify-between items-center gap-4 text-sm text-muted-foreground'>
          <p>© 2026 Modern Platform. All rights reserved.</p>
          <div className='flex gap-8'>
            <Link
              href='/docs/privacy-policy'
              className='hover:text-foreground transition-colors'
            >
              {t('privacyPolicy')}
            </Link>
            <Link
              href='/docs/terms-of-service'
              className='hover:text-foreground transition-colors'
            >
              {t('termsOfService')}
            </Link>
          </div>
        </div>
      </div>

      <div className='absolute bottom-0 left-0 w-full overflow-hidden pointer-events-none opacity-[0.02]'>
        <div className='text-[12vw] 2xl:text-[180px] font-black leading-none text-foreground whitespace-nowrap select-none text-center transform translate-y-1/3 transition-all duration-700'>
          Modern Platform
        </div>
      </div>
    </footer>
  );
});

function SocialLink({ icon: Icon, href }: { icon: LucideIcon; href: string }) {
  return (
    <Link
      href={href}
      className='w-10 h-10 rounded-full bg-muted/50 flex items-center justify-center text-muted-foreground hover:bg-primary hover:text-primary-foreground transition-all duration-300'
    >
      <Icon className='w-5 h-5' />
    </Link>
  );
}

function FooterLink({
  href,
  children,
}: {
  href: string;
  children: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      className='hover:text-foreground transition-colors flex items-center group'
    >
      <span className='relative'>
        {children}
        <span className='absolute left-0 -bottom-0.5 w-0 h-px bg-foreground transition-all duration-300 group-hover:w-full' />
      </span>
    </Link>
  );
}
