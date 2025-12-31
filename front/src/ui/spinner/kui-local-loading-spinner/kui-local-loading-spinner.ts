import { Component, input } from '@angular/core';
import { ProgressSpinner } from 'primeng/progressspinner';

@Component({
  selector: 'kui-local-loading-spinner',
  imports: [ProgressSpinner],
  templateUrl: './kui-local-loading-spinner.html',
  styleUrl: './kui-local-loading-spinner.css',
})
export class KuiLocalLoadingSpinner {
  visible = input<boolean>(false);
  message = input<string>('');
  size = input<string>('16');
  strokeWidth = input<string>('4');
  animationDuration = input<string>('1s');
}
