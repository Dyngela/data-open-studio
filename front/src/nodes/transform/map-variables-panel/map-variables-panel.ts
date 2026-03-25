import { Component, input, model, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MapVariable, VariableKind } from '../definition';

export interface VariableInputSchema {
  name: string;
  schema: { name: string; type: string }[];
}

@Component({
  selector: 'app-map-variables-panel',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './map-variables-panel.html',
  styleUrl: './map-variables-panel.css',
})
export class MapVariablesPanel {
  variables = model<MapVariable[]>([]);
  inputs = input<VariableInputSchema[]>([]);

  protected draggedVarIndex = signal<number | null>(null);
  protected dropTargetIndex = signal<number | null>(null);

  addVariable(kind: VariableKind = 'computed') {
    const name = this.generateName(kind);
    this.variables.update(vars => [
      ...vars,
      { name, kind, expression: '', dataType: 'string' },
    ]);
  }

  removeVariable(index: number) {
    this.variables.update(vars => vars.filter((_, i) => i !== index));
  }

  updateVariable(index: number, field: keyof MapVariable, value: string) {
    this.variables.update(vars => {
      const next = [...vars];
      next[index] = { ...next[index], [field]: value };
      return next;
    });
  }

  toggleKind(index: number) {
    this.variables.update(vars => {
      const next = [...vars];
      const current = next[index];
      next[index] = {
        ...current,
        kind: current.kind === 'filter' ? 'computed' : 'filter',
      };
      return next;
    });
  }

  // Insert a column reference at the end of a variable's expression
  insertRef(varIndex: number, inputName: string, colName: string) {
    const ref = `${inputName}.${colName}`;
    this.variables.update(vars => {
      const next = [...vars];
      const current = next[varIndex].expression;
      next[varIndex] = {
        ...next[varIndex],
        expression: current ? `${current} ${ref}` : ref,
      };
      return next;
    });
  }

  // Handle drop of an input column onto a variable's expression textarea
  onExpressionDrop(event: DragEvent, varIndex: number) {
    event.preventDefault();
    event.stopPropagation();
    const data = event.dataTransfer?.getData('text/plain');
    if (!data) return;

    this.variables.update(vars => {
      const next = [...vars];
      const current = next[varIndex].expression;
      next[varIndex] = {
        ...next[varIndex],
        expression: current ? `${current} ${data}` : data,
      };
      return next;
    });
  }

  onExpressionDragOver(event: DragEvent) {
    event.preventDefault();
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy';
  }

  // Drag start for computed variables (to drag to output panel)
  onVarDragStart(event: DragEvent, index: number) {
    const v = this.variables()[index];
    if (v.kind !== 'computed') return;

    this.draggedVarIndex.set(index);
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'copyMove';
      event.dataTransfer.setData('text/plain', `$var.${v.name}`);
      event.dataTransfer.setData('application/x-var-name', v.name);
      event.dataTransfer.setData('application/x-var-type', v.dataType);
    }
  }

  onVarDragEnd() {
    this.draggedVarIndex.set(null);
    this.dropTargetIndex.set(null);
  }

  // Reorder: drag over another variable row
  onRowDragOver(event: DragEvent, index: number) {
    if (this.draggedVarIndex() === null) return;
    event.preventDefault();
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
    this.dropTargetIndex.set(index);
  }

  onRowDragLeave() {
    this.dropTargetIndex.set(null);
  }

  onRowDrop(event: DragEvent, targetIndex: number) {
    event.preventDefault();
    const fromIndex = this.draggedVarIndex();
    if (fromIndex === null || fromIndex === targetIndex) return;

    this.variables.update(vars => {
      const next = [...vars];
      const [moved] = next.splice(fromIndex, 1);
      next.splice(targetIndex, 0, moved);
      return next;
    });

    this.draggedVarIndex.set(null);
    this.dropTargetIndex.set(null);
  }

  moveUp(index: number) {
    if (index === 0) return;
    this.variables.update(vars => {
      const next = [...vars];
      [next[index - 1], next[index]] = [next[index], next[index - 1]];
      return next;
    });
  }

  moveDown(index: number) {
    const vars = this.variables();
    if (index >= vars.length - 1) return;
    this.variables.update(v => {
      const next = [...v];
      [next[index], next[index + 1]] = [next[index + 1], next[index]];
      return next;
    });
  }

  private generateName(kind: VariableKind): string {
    const prefix = kind === 'filter' ? 'filter' : 'var';
    const existing = this.variables().map(v => v.name);
    let i = 1;
    while (existing.includes(`${prefix}_${i}`)) i++;
    return `${prefix}_${i}`;
  }
}
