import { Component, input, output, signal, inject } from '@angular/core';

@Component({
  selector: 'app-transform-modal',
  standalone: true,
  imports: [],
  templateUrl: './transform.modal.html',
  styleUrl: './transform.modal.css',
})
export class TransformModal {
    close = output<void>();


  onCancel() {
    this.close.emit();
  }
}
