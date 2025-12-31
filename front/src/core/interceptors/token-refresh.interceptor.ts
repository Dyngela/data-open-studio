import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, switchMap, throwError } from 'rxjs';
import { AuthService } from '../api/auth.service';

/**
 * Token refresh interceptor
 * Automatically refreshes the access token when receiving a 401 error
 */
export const tokenRefreshInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      // Only handle 401 errors and if not already on auth pages
      if (error.status === 401 && !req.url.includes('/auth/')) {
        const refreshToken = authService.getRefreshToken();

        if (refreshToken && !req.url.includes('/auth/refresh')) {
          // Try to refresh the token
          return authService.refreshToken().pipe(
            switchMap(() => {
              // Retry the original request with new token
              const token = authService.getAccessToken();
              const clonedRequest = req.clone({
                setHeaders: {
                  Authorization: `Bearer ${token}`
                }
              });
              return next(clonedRequest);
            }),
            catchError((refreshError) => {
              // Refresh failed, logout and redirect
              authService.logout();
              router.navigate(['/auth/login']);
              return throwError(() => refreshError);
            })
          );
        } else {
          // No refresh token available, logout
          authService.logout();
          router.navigate(['/auth/login']);
        }
      }

      return throwError(() => error);
    })
  );
};
