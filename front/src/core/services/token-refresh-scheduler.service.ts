import { Injectable, inject, DestroyRef } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { interval } from 'rxjs';
import { AuthService } from '../api/auth.service';
import { jwtDecode } from 'jwt-decode';

interface JwtPayload {
  exp: number;
  iat: number;
  userId: number;
}

/**
 * Token refresh scheduler service
 * Automatically refreshes the access token before it expires
 */
@Injectable({
  providedIn: 'root'
})
export class TokenRefreshSchedulerService {
  private authService = inject(AuthService);
  private destroyRef = inject(DestroyRef);

  /**
   * Start automatic token refresh
   * Checks every minute if token needs refresh
   */
  startRefreshScheduler(): void {
    // Check every 3 minutes
    interval(180_000)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => {
        this.checkAndRefreshToken();
      });

    this.checkAndRefreshToken();
  }

  /**
   * Check if token needs refresh and refresh if necessary
   */
  private checkAndRefreshToken(): void {
    const token = this.authService.getAccessToken();
    const refreshToken = this.authService.getRefreshToken();

    if (!token || !refreshToken) {
      return;
    }

    try {
      const decoded = jwtDecode<JwtPayload>(token);
      const expirationTime = decoded.exp * 1000; // Convert to milliseconds
      const currentTime = Date.now();
      const timeUntilExpiry = expirationTime - currentTime;

      const fiveMinutes = 5 * 60 * 1000;

      if (timeUntilExpiry < fiveMinutes && timeUntilExpiry > 0) {
        this.authService.refreshToken().subscribe({
          next: () => {
          },
          error: (error) => {
            console.error('Failed to refresh token:', error);
            this.authService.logout();
          }
        });
      }
    } catch (error) {
      console.error('Error decoding token:', error);
    }
  }
}
