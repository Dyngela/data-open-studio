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
import {MapGlobalFilter} from './map-global-filter/map-global-filter';
import {MapOutputField} from './map-output-field/map-output-field';

interface JoinKeyPair {
  leftCol: string;
  rightCol: string;
}

type JoinType = 'inner' | 'left' | 'right';

@Component({
  selector: 'app-transform-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, MapGlobalFilter, MapOutputField],
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

  protected downstreamSchema = computed(() => {
    return this.jobState.getDownstreamExpectedSchema(this.node().id);
  });

  protected outputColumns = signal<MapOutputCol[]>([]);
  protected joinType = signal<JoinType>('inner');
  protected joinKeys = signal<JoinKeyPair[]>([]);

  protected draggedColumn = signal<{ inputName: string; colName: string; colType: string } | null>(null);
  protected dropTargetCol = signal<string | null>(null);
  protected dropTargetOutputIndex = signal<number | null>(null);
  protected outputPanelHovered = signal(false);

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
      event.dataTransfer.effectAllowed = 'copyLink';
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
    this.dropTargetOutputIndex.set(null);
    this.outputPanelHovered.set(false);
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

  // ── Drag & drop: input → output mapping ────────────────

  onOutputPanelDragOver(event: DragEvent) {
    if (!this.draggedColumn()) return;
    event.preventDefault();
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy';
    this.outputPanelHovered.set(true);
  }

  onOutputPanelDragLeave(event: DragEvent) {
    const related = event.relatedTarget as HTMLElement;
    if (related && (event.currentTarget as HTMLElement).contains(related)) return;
    this.outputPanelHovered.set(false);
  }

  onOutputPanelDrop(event: DragEvent) {
    event.preventDefault();
    const dragged = this.draggedColumn();
    if (!dragged) return;

    // Add as a new output column (panel-level drop, not on a specific item)
    this.addColumn(dragged.inputName, { name: dragged.colName, type: dragged.colType });

    this.draggedColumn.set(null);
    this.outputPanelHovered.set(false);
  }

  onOutputItemDragOver(event: DragEvent, index: number) {
    if (!this.draggedColumn()) return;
    event.preventDefault();
    event.stopPropagation();
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'link';
    this.dropTargetOutputIndex.set(index);
  }

  onOutputItemDragLeave() {
    this.dropTargetOutputIndex.set(null);
  }

  onOutputItemDrop(event: DragEvent, index: number) {
    event.preventDefault();
    event.stopPropagation();
    const dragged = this.draggedColumn();
    if (!dragged) return;

    // Map the dragged input column onto this output column
    this.outputColumns.update(cols => {
      const next = [...cols];
      next[index] = {
        ...next[index],
        inputRef: `${dragged.inputName}.${dragged.colName}`,
      };
      return next;
    });

    this.draggedColumn.set(null);
    this.dropTargetOutputIndex.set(null);
    this.outputPanelHovered.set(false);
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

  /** Load the downstream target schema (names + types only, no mapping). */
  fillFromDownstream() {
    const downstream = this.downstreamSchema();
    if (!downstream.length) return;

    const columns: MapOutputCol[] = downstream.map(col => ({
      name: col.name,
      dataType: col.type,
      funcType: 'direct' as const,
      inputRef: '',
    }));

    this.outputColumns.set(columns);
  }

  /** Auto-match unmapped output columns to upstream inputs by column name. */
  autoMatchColumns() {
    const inputs = this.upstreamInputs();
    if (!inputs.length) return;

    // Track matches and whether they are unique
    // Value will be the match object, or null if a duplicate is found
    const upstreamLookup = new Map<string, { inputName: string; colName: string } | null>();

    for (const input of inputs) {
      for (const col of input.schema) {
        const key = col.name.toLowerCase();

        if (upstreamLookup.has(key)) {
          // If it already exists, mark it as a duplicate (null)
          upstreamLookup.set(key, null);
        } else {
          // First time seeing this name
          upstreamLookup.set(key, { inputName: input.name, colName: col.name });
        }
      }
    }

    this.outputColumns.update(cols =>
      cols.map(col => {
        if (col.inputRef) return col;

        const match = upstreamLookup.get(col.name.toLowerCase());

        // Only match if we found it exactly once (match is not null)
        if (match) {
          return { ...col, inputRef: `${match.inputName}.${match.colName}` };
        }

        return col;
      }),
    );
  }

  addAllColumns(inputName: string, schema: DataModel[]) {
    for (const col of schema) {
      this.addColumn(inputName, col);
    }
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
