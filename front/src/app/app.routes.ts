import { Routes } from '@angular/router';
import { authGuard, guestGuard } from '../core/guards/auth.guard';
import { Playground } from '../views/graph/playground/playground';
import { Triggers } from '../views/triggers/triggers/triggers';
import { Settings } from '../views/settings/settings/settings';
import { Jobs } from '../views/jobs/jobs/jobs';
import { Login } from '../views/authentication/login/login';
import { Register } from '../views/authentication/register/register';

export const routes: Routes = [
  // Auth routes (accessible only when not logged in)
  {
    path: 'auth',
    canActivate: [guestGuard],
    children: [
      { path: '', redirectTo: 'login', pathMatch: 'full' },
      { path: 'login', component: Login },
      { path: 'register', component: Register },
    ]
  },

  // Protected routes (require authentication)
  { path: '', redirectTo: 'jobs', pathMatch: 'full' },
  { path: 'playground', component: Playground, canActivate: [authGuard] },
  { path: 'playground/:id', component: Playground, canActivate: [authGuard] },
  { path: 'triggers', component: Triggers, canActivate: [authGuard] },
  {
    path: 'settings',
    component: Settings,
    canActivate: [authGuard],
    children: [
      { path: '', redirectTo: 'db', pathMatch: 'full' },
      {
        path: 'db',
        loadComponent: () => import('../views/settings/db-metadata-list/db-metadata-list').then(m => m.DbMetadataList)
      },
      {
        path: 'sftp',
        loadComponent: () => import('../views/settings/sftp-metadata-list/sftp-metadata-list').then(m => m.SftpMetadataList)
      },
      {
        path: 'email',
        loadComponent: () => import('../views/settings/email-metadata-list/email-metadata-list').then(m => m.EmailMetadataList)
      }
    ]
  },
  { path: 'jobs', component: Jobs, canActivate: [authGuard] },

  // Fallback
  { path: '**', redirectTo: 'jobs' }
];
