import { Component, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, RouterLink, RouterLinkActive, Router } from '@angular/router';
import { AuthService } from '../core/api/auth.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, RouterLinkActive],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  private authService = inject(AuthService);
  private router = inject(Router);

  currentUser = this.authService.currentUser;
  isAuthenticated = computed(() => {
    console.log('Auth check:', this.authService.isAuthenticated());
    return this.authService.isAuthenticated()
  });

  userInitials = computed(() => {
    const user = this.currentUser();
    if (!user) return 'U';
    const first = user.prenom?.charAt(0) || '';
    const last = user.nom?.charAt(0) || '';
    return (first + last).toUpperCase() || 'U';
  });

  logout() {
    this.authService.logout();
    this.router.navigate(['/auth/login']);
  }
}
