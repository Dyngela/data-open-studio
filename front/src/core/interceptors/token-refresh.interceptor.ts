import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import {BehaviorSubject, catchError, filter, from, switchMap, take, throwError} from 'rxjs';
import { AuthService } from '../api/auth.service';

/**
 * Token refresh interceptor
 * Automatically refreshes the access token when receiving a 401 error
 */
let isRefreshing = false;
const refreshTokenSubject = new BehaviorSubject<string | null>(null);

export const tokenRefreshInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  if (req.url.includes('/auth/')) {
    return next(req);
  }

  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      if (error.status !== 401) {
        return throwError(() => error);
      }

      const refreshToken = authService.getRefreshToken();

      if (!refreshToken) {
        authService.logout();
        router.navigate(['/auth/login']);
        return throwError(() => error);
      }

      if (isRefreshing) {
        return refreshTokenSubject.pipe(
          filter(token => token !== null),
          take(1),
          switchMap(token => next(req.clone({
            setHeaders: { Authorization: `Bearer ${token}` }
          })))
        );
      }

      isRefreshing = true;
      refreshTokenSubject.next(null);

      return authService.refreshToken().pipe(
        switchMap(() => {
          const token = authService.getAccessToken()
          isRefreshing = false;
          refreshTokenSubject.next(token);

          return next(req.clone({
            setHeaders: { Authorization: `Bearer ${token}` }
          }));
        }),
        catchError(refreshError => {
          isRefreshing = false;
          authService.logout();
          router.navigate(['/auth/login']);
          return throwError(() => refreshError);
        })
      );
    })
  );
};
