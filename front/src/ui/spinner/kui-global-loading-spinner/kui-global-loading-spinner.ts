import { Component, input } from '@angular/core';
import { ProgressSpinner } from 'primeng/progressspinner';

@Component({
  selector: 'kui-global-loading-spinner',
  imports: [ProgressSpinner],
  templateUrl: './kui-global-loading-spinner.html',
  styleUrl: './kui-global-loading-spinner.scss',
})
export class KuiGlobalLoadingSpinner {
  visible = input<boolean>(false);
  message = input<string>('');
  strokeWidth = input<string>('4');
  animationDuration = input<string>('1s');
}
