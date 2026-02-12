import {ApplicationConfig, inject, provideAppInitializer} from '@angular/core';
import { provideRouter } from '@angular/router';

import { routes } from './app.routes';
import {provideHttpClient, withInterceptors} from '@angular/common/http';
import {provideAnimationsAsync} from '@angular/platform-browser/animations/async';
import {PrimeNG, providePrimeNG} from 'primeng/config';
import {ConfirmationService, MessageService} from 'primeng/api';
import Aura from '@primeng/themes/aura';
import {authInterceptor} from '../core/interceptors/auth.interceptor';
import {tokenRefreshInterceptor} from '../core/interceptors/token-refresh.interceptor';
import {PrimeLocaleService} from '../core/services/prime-locale.service';


export const appConfig: ApplicationConfig = {
  providers: [
    provideRouter(routes),
    provideHttpClient(
      withInterceptors([
        authInterceptor,
        tokenRefreshInterceptor,
      ])
    ),
    provideAnimationsAsync(),
    providePrimeNG({
      theme: {
        preset: Aura,
      },
      ripple: true,
      zIndex: {
        modal: 4000,
        overlay: 5000,
        menu: 5000,
        tooltip: 6000,
      },
    }),
    PrimeLocaleService,
    provideAppInitializer(() => {
      const primeNG = inject(PrimeNG);
      const localeService = inject(PrimeLocaleService);
      const locale = localStorage.getItem('locale') ?? navigator.language.split('-')[0] ?? 'fr';

      localeService.setLocale(primeNG, locale);
    }),
    MessageService,
    ConfirmationService,
  ]
};
