import { Component, input, output, signal, inject } from '@angular/core';
import {LayoutService} from '../../core/services/layout-service';

@Component({
  selector: 'app-start-modal',
  standalone: true,
  imports: [],
  templateUrl: './start.modal.html',
  styleUrl: './start.modal.css',
})
export class StartModal {
  private layout = inject(LayoutService);

  onCancel() {
    this.layout.closeModal();
  }
}
