import { Component, input, output } from '@angular/core';
import { Button } from 'primeng/button';

@Component({
  selector: 'kui-submit-button',
  imports: [Button],
  templateUrl: './kui-submit-button.html',
  styleUrl: './kui-submit-button.css',
})
export class KuiSubmitButton {
  label = input<string>('Valider');
  icon = input<string>('pi pi-check');
  loading = input<boolean>(false);
  disabled = input<boolean>(false);
  customClass = input<string>('');

  onClick = output<Event>();

  handleClick(event: Event): void {
    this.onClick.emit(event);
  }
}
