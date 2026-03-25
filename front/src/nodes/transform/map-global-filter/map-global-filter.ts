import { Component, input, model } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

export interface FilterInputSchema {
  name: string;
  schema: { name: string; type: string }[];
}

@Component({
  selector: 'app-map-global-filter',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './map-global-filter.html',
  styleUrl: './map-global-filter.css',
})
export class MapGlobalFilter {
  model = model<string>('');
  inputs = input<FilterInputSchema[]>([]);

  insertRef(inputName: string, colName: string) {
    const ref = `${inputName}.${colName}`;
    const current = this.model();
    this.model.set(current ? `${current} ${ref}` : ref);
  }
}
