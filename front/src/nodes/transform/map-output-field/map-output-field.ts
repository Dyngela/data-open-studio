import {Component, input, Input, WritableSignal} from '@angular/core';
import {MapOutputCol} from '../definition';

@Component({
  selector: 'app-map-output-field',
  imports: [],
  templateUrl: './map-output-field.html',
  styleUrl: './map-output-field.css',
})
export class MapOutputField {
  column = input.required<MapOutputCol>();
  index = input.required<number>();
  columns = input.required<WritableSignal<MapOutputCol[]>>();

  clearMapping(index: number) {
    this.columns().update(cols => {
      const next = [...cols];
      next[index] = { ...next[index], inputRef: '' };
      return next;
    });
  }

  removeColumn(index: number) {
    this.columns().update(cols => cols.filter((_, i) => i !== index));
  }

  moveUp(index: number) {
    if (index <= 0) return;
    this.columns().update(cols => {
      const next = [...cols];
      [next[index - 1], next[index]] = [next[index], next[index - 1]];
      return next;
    });
  }

  moveDown(index: number) {
    const cols = this.columns();
    if (index >= cols().length - 1) return;
    this.columns().update(cols => {
      const next = [...cols];
      [next[index], next[index + 1]] = [next[index + 1], next[index]];
      return next;
    });
  }
}
