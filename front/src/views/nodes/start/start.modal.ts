import { Component, input, output, signal, inject } from '@angular/core';

@Component({
  selector: 'app-start-modal',
  standalone: true,
  imports: [],
  templateUrl: './start.modal.html',
  styleUrl: './start.modal.css',
})
export class StartModal {
  close = output<void>();

  onCancel() {
    this.close.emit();
  }
}
