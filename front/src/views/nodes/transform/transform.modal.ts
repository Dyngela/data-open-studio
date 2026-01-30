import { Component, input, output, signal, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NodeInstance } from '../../../core/nodes-services/node.type';
import { JobStateService } from '../../../core/nodes-services/job-state.service';
import {
  InputFlow,
  MapNodeConfig,
  MapOutputCol,
  OutputFlow,
  isMapConfig,
} from '../../../core/nodes-services/node-configs.type';

@Component({
  selector: 'app-transform-modal',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './transform.modal.html',
  styleUrl: './transform.modal.css',
})
export class TransformModal {
  node = input.required<NodeInstance>();
  close = output<void>();
  save = output<MapNodeConfig>();

  private jobState = inject(JobStateService);

  /** Upstream inputs with their schemas */
  protected upstreamInputs = computed(() => {
    const n = this.node();
    return this.jobState.getUpstreamSchemas(n.id);
  });

  /** Output columns being configured */
  protected outputColumns = signal<MapOutputCol[]>([]);

  ngOnInit() {
    // Restore existing config if any
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isMapConfig(config) && config.outputs?.length) {
      this.outputColumns.set([...config.outputs[0].columns]);
    }
  }

  /** Add an upstream column as a direct mapping to the output */
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

  /** Remove an output column by index */
  removeColumn(index: number) {
    this.outputColumns.update(cols => cols.filter((_, i) => i !== index));
  }

  /** Move a column up in the output list */
  moveUp(index: number) {
    if (index <= 0) return;
    this.outputColumns.update(cols => {
      const next = [...cols];
      [next[index - 1], next[index]] = [next[index], next[index - 1]];
      return next;
    });
  }

  /** Move a column down in the output list */
  moveDown(index: number) {
    const cols = this.outputColumns();
    if (index >= cols.length - 1) return;
    this.outputColumns.update(cols => {
      const next = [...cols];
      [next[index], next[index + 1]] = [next[index + 1], next[index]];
      return next;
    });
  }

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

    this.save.emit(config);
  }

  onCancel() {
    this.close.emit();
  }
}
