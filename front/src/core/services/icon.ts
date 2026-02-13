import {Component, computed, inject, input} from '@angular/core';
import {IconRegistryService} from './icon-registry-service';

@Component({
  selector: 'app-icon',
  standalone: true,
  template: `
    <svg
      [attr.viewBox]="viewBox()"
      [attr.width]="size()"
      [attr.height]="size()"
      fill="currentColor">
      <path [attr.d]="path()" />
    </svg>
  `,
  styles: `
    :host { display: inline-flex; align-items: center; justify-content: center; }
  `
})
export class Icon {
  private registry = inject(IconRegistryService);

  name = input.required<string>();
  size = input<number | string>(24);
  viewBox = input<string>('0 0 24 24');

  path = computed(() => this.registry.getIcon(this.name()));
}
