import { Routes } from '@angular/router';
import { Playground } from '../views/graph/playground/playground';

export const routes: Routes = [
  { path: '', redirectTo: 'playground', pathMatch: 'full' },
  { path: 'playground', component: Playground },
];
