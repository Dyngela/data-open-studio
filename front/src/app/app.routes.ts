import { Routes } from '@angular/router';
import { Playground } from '../views/graph/playground/playground';
import {Triggers} from '../views/triggers/triggers/triggers';
import {Settings} from '../views/settings/settings/settings';
import {Jobs} from '../views/jobs/jobs/jobs';

export const routes: Routes = [
  { path: '', redirectTo: 'playground', pathMatch: 'full' },
  { path: 'playground', component: Playground },
  { path: 'triggers', component: Triggers },
  { path: 'settings', component: Settings },
  { path: 'jobs', component: Jobs},

];
