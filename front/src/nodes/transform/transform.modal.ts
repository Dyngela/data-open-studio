import { Component, input, output, signal, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { JobStateService } from '../../core/nodes-services/job-state.service';
import {
  InputFlow,
  MapNodeConfig,
  MapOutputCol,
  OutputFlow,
  isMapConfig,
} from './definition';
import { LayoutService } from '../../core/services/layout-service';
import { DataModel } from '../../core/api/metadata.type';

interface JoinKeyPair {
  leftCol: string;
  rightCol: string;
}

type JoinType = 'inner' | 'left' | 'right';

@Component({
  selector: 'app-transform-modal',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './transform.modal.html',
  styleUrl: './transform.modal.css',
})
export class TransformModal {
  private layoutService = inject(LayoutService);
  node = input.required<NodeInstance>();
  save = output<MapNodeConfig>();

  private jobState = inject(JobStateService);

  protected upstreamInputs = computed(() => {
    return this.jobState.getUpstreamSchemas(this.node().id);
  });

  protected hasMultipleInputs = computed(() => this.upstreamInputs().length >= 2);
  protected leftInput = computed(() => this.upstreamInputs()[0] ?? null);
  protected rightInput = computed(() => this.upstreamInputs()[1] ?? null);

  protected outputColumns = signal<MapOutputCol[]>([]);
  protected joinType = signal<JoinType>('inner');
  protected joinKeys = signal<JoinKeyPair[]>([]);

  protected draggedColumn = signal<{ inputName: string; colName: string; colType: string } | null>(null);
  protected dropTargetCol = signal<string | null>(null);

  ngOnInit() {
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isMapConfig(config)) {
      if (config.outputs?.length) {
        this.outputColumns.set([...config.outputs[0].columns]);
      }
      if (config.join) {
        this.joinType.set(config.join.type as JoinType);
        const pairs: JoinKeyPair[] = config.join.leftKeys.map((lk, i) => ({
          leftCol: lk,
          rightCol: config.join!.rightKeys[i],
        }));
        this.joinKeys.set(pairs);
      }
    }
  }

  // ── Drag & drop for join keys ──────────────────────────

  onDragStart(event: DragEvent, inputName: string, col: DataModel) {
    this.draggedColumn.set({ inputName, colName: col.name, colType: col.type });
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'link';
      event.dataTransfer.setData('text/plain', `${inputName}.${col.name}`);
    }
  }

  onDragOver(event: DragEvent, targetInputName: string, targetColName: string) {
    const dragged = this.draggedColumn();
    if (!dragged || dragged.inputName === targetInputName) return;
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'link';
    }
    this.dropTargetCol.set(`${targetInputName}.${targetColName}`);
  }

  onDragLeave() {
    this.dropTargetCol.set(null);
  }

  onDrop(event: DragEvent, targetInputName: string, targetCol: DataModel) {
    event.preventDefault();
    const dragged = this.draggedColumn();

    // 1. Safety checks
    if (!dragged || dragged.inputName === targetInputName) return;
    const left = this.leftInput();
    const right = this.rightInput();
    if (!left || !right) return;

    // 2. Explicitly assign based on input identity, not drag direction
    let leftCol: string = '';
    let rightCol: string = '';

    // Determine which piece of data belongs to 'Left' and which to 'Right'
    if (dragged.inputName === left.name) {
      leftCol = dragged.colName;
      rightCol = targetCol.name; // Because target must be right
    } else if (dragged.inputName === right.name) {
      rightCol = dragged.colName;
      leftCol = targetCol.name; // Because target must be left
    }

    // 3. Update state
    const exists = this.joinKeys().some(k => k.leftCol === leftCol && k.rightCol === rightCol);
    if (!exists && leftCol && rightCol) {
      this.joinKeys.update(keys => [...keys, { leftCol, rightCol }]);
    }

    this.draggedColumn.set(null);
    this.dropTargetCol.set(null);
  }

  onDragEnd() {
    this.draggedColumn.set(null);
    this.dropTargetCol.set(null);
  }

  removeJoinKey(index: number) {
    this.joinKeys.update(keys => keys.filter((_, i) => i !== index));
  }

  setJoinType(type: JoinType) {
    this.joinType.set(type);
  }

  isDropTarget(inputName: string, colName: string): boolean {
    return this.dropTargetCol() === `${inputName}.${colName}`;
  }

  isJoinedCol(inputName: string, colName: string): boolean {
    const left = this.leftInput();
    if (!left) return false;
    if (inputName === left.name) {
      return this.joinKeys().some(k => k.leftCol === colName);
    }
    return this.joinKeys().some(k => k.rightCol === colName);
  }

  // ── Output columns ────────────────────────────────────

  addColumn(inputName: string, col: { name: string; type: string }) {
    const existing = this.outputColumns();
    const alreadyExists = existing.some(
      c => c.funcType === 'direct' && c.inputRef === `${inputName}.${col.name}`,
    );
    if (alreadyExists) return;

    this.outputColumns.update(cols => [
      ...cols,
      {
        name: col.name,
        dataType: col.type,
        funcType: 'direct' as const,
        inputRef: `${inputName}.${col.name}`,
      },
    ]);
  }

  addAllColumns(inputName: string, schema: DataModel[]) {
    for (const col of schema) {
      this.addColumn(inputName, col);
    }
  }

  removeColumn(index: number) {
    this.outputColumns.update(cols => cols.filter((_, i) => i !== index));
  }

  moveUp(index: number) {
    if (index <= 0) return;
    this.outputColumns.update(cols => {
      const next = [...cols];
      [next[index - 1], next[index]] = [next[index], next[index - 1]];
      return next;
    });
  }

  moveDown(index: number) {
    const cols = this.outputColumns();
    if (index >= cols.length - 1) return;
    this.outputColumns.update(cols => {
      const next = [...cols];
      [next[index], next[index + 1]] = [next[index + 1], next[index]];
      return next;
    });
  }

  // ── Save / Cancel ─────────────────────────────────────

  onSave() {
    const inputs: InputFlow[] = this.upstreamInputs().map(up => ({
      name: up.name,
      portId: up.portId,
      schema: up.schema,
    }));

    const outputFlow: OutputFlow = {
      name: 'out',
      portId: 0,
      columns: this.outputColumns(),
    };

    const config: MapNodeConfig = {
      kind: 'map',
      inputs,
      outputs: [outputFlow],
    };

    if (this.hasMultipleInputs() && this.joinKeys().length > 0) {
      const left = this.leftInput()!;
      const right = this.rightInput()!;
      config.join = {
        type: this.joinType(),
        leftInput: left.name,
        rightInput: right.name,
        leftKeys: this.joinKeys().map(k => k.leftCol),
        rightKeys: this.joinKeys().map(k => k.rightCol),
      };
    }

    this.jobState.setNodeConfig(this.node().id, config);
    this.layoutService.closeModal();
  }

  onCancel() {
    this.layoutService.closeModal();
  }
}
