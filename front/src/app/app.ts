import {Component, inject, computed, effect, OnInit, AfterViewInit, HostListener, signal, untracked} from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, RouterLink, RouterLinkActive, Router } from '@angular/router';
import { AuthService } from '../core/api/auth.service';
import {MetadataLocalService} from '../core/services/metadata.local.service';
import {Toast} from 'primeng/toast';
import {Button} from 'primeng/button';
import {NodePanel} from '../views/graph/node-panel/node-panel';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, RouterLinkActive, Toast],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App implements AfterViewInit {
  private authService = inject(AuthService);
  private localMetadata = inject(MetadataLocalService)
  private router = inject(Router);

  currentUser = this.authService.currentUser;
  isAuthenticated = this.authService.isAuthenticated;

  constructor() {
    effect(() => {
      if (this.authService.isAuthenticated()){
        untracked(() => this.localMetadata.initialize())
      }
    });
  }

  ngAfterViewInit() {
    this.authService.initializeAuth();
  }

  userInitials = computed(() => {
    const user = this.currentUser();
    if (!user || !this.isAuthenticated()) return 'U';
    const first = user.prenom?.charAt(0) || '';
    const last = user.nom?.charAt(0) || '';
    return (first + last).toUpperCase() || 'U';
  });

  logout() {
    this.authService.logout();
    this.router.navigate(['/auth/login']);
  }



}
