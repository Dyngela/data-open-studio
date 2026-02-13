import { Component, input, output } from '@angular/core';

@Component({
  selector: 'kui-modal-header',
  standalone: true,
  templateUrl: './kui-modal-header.html',
  styleUrl: './kui-modal-header.css',
})
export class KuiModalHeader {
  title = input.required<string>();
  subtitle = input<string>('');
  closeLabel = input<string>('Fermer');
  editable = input<boolean>(false);

  close = output<void>();
  titleChange = output<string>();

  onClose(): void {
    this.close.emit();
  }

  onTitleInput(value: string): void {
    this.titleChange.emit(value);
  }
}
