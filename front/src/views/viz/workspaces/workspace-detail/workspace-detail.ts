import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { Button } from 'primeng/button';
import { TableModule } from 'primeng/table';
import { Tag } from 'primeng/tag';
import { ProgressSpinner } from 'primeng/progressspinner';
import { PanelModule } from 'primeng/panel';
import { Toast } from 'primeng/toast';
import { ConfirmDialog } from 'primeng/confirmdialog';
import { ConfirmationService, MessageService } from 'primeng/api';

import { VizService } from '../../../../core/api/viz.service';
import { FrameSchema, FrameData, Workspace } from '../../../../core/api/viz.type';

@Component({
  selector: 'app-workspace-detail',
  standalone: true,
  imports: [
    CommonModule,
    Button,
    TableModule,
    Tag,
    ProgressSpinner,
    PanelModule,
    Toast,
    ConfirmDialog,
  ],
  providers: [MessageService, ConfirmationService],
  templateUrl: './workspace-detail.html',
})
export class WorkspaceDetail implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private vizService = inject(VizService);
  private messageService = inject(MessageService);
  private confirmationService = inject(ConfirmationService);

  workspaceId = signal('');

  workspaceResult = signal<ReturnType<typeof this.vizService.getWorkspace> | null>(null);
  workspace = computed<Workspace | null>(() => this.workspaceResult()?.data() ?? null);
  isLoadingWorkspace = computed(() => this.workspaceResult()?.isLoading() ?? true);

  framesResult = signal<ReturnType<typeof this.vizService.listFrames> | null>(null);
  frames = computed<FrameSchema[]>(() => this.framesResult()?.data()?.frames ?? []);
  isLoadingFrames = computed(() => this.framesResult()?.isLoading() ?? true);

  // Per-frame preview state keyed by frame name
  previewResults = signal<Record<string, FrameData | null>>({});
  previewLoading = signal<Record<string, boolean>>({});

  expandedFrames = signal<Set<string>>(new Set());

  ngOnInit() {
    const id = this.route.snapshot.paramMap.get('id') ?? '';
    this.workspaceId.set(id);

    const wsResult = this.vizService.getWorkspace(id);
    this.workspaceResult.set(wsResult);

    const frResult = this.vizService.listFrames(id);
    this.framesResult.set(frResult);
  }

  refresh() {
    this.framesResult()?.refresh();
  }

  togglePreview(frame: FrameSchema) {
    const name = frame.name;
    const expanded = new Set(this.expandedFrames());

    if (expanded.has(name)) {
      expanded.delete(name);
      this.expandedFrames.set(expanded);
      return;
    }

    expanded.add(name);
    this.expandedFrames.set(expanded);

    // Only fetch if not already loaded
    if (this.previewResults()[name] !== undefined) return;

    this.previewLoading.update(m => ({ ...m, [name]: true }));

    const result = this.vizService.previewFrame(
      this.workspaceId(),
      name,
      0,
      100,
      (data) => {
        this.previewResults.update(m => ({ ...m, [name]: data }));
        this.previewLoading.update(m => ({ ...m, [name]: false }));
      },
    );
    // Trigger it (fetch runs immediately inside vizService.fetch)
    void result;
  }

  isExpanded(name: string): boolean {
    return this.expandedFrames().has(name);
  }

  previewData(name: string): FrameData | null {
    return this.previewResults()[name] ?? null;
  }

  isPreviewLoading(name: string): boolean {
    return this.previewLoading()[name] ?? false;
  }

  previewColumns(data: FrameData): string[] {
    return Object.keys(data.columns);
  }

  previewRows(data: FrameData): Record<string, unknown>[] {
    const cols = this.previewColumns(data);
    const len = cols.length > 0 ? (data.columns[cols[0]]?.length ?? 0) : 0;
    const rows: Record<string, unknown>[] = [];
    for (let i = 0; i < len; i++) {
      const row: Record<string, unknown> = {};
      for (const col of cols) {
        row[col] = data.columns[col][i];
      }
      rows.push(row);
    }
    return rows;
  }

  dtypeSeverity(dtype: string): 'info' | 'secondary' | 'success' | 'warn' {
    if (['Int8','Int16','Int32','Int64','UInt8','UInt16','UInt32','UInt64'].includes(dtype)) return 'info';
    if (['Float32','Float64'].includes(dtype)) return 'warn';
    if (['Boolean'].includes(dtype)) return 'success';
    if (['Date','Datetime','Time','Duration'].includes(dtype)) return 'warn';
    return 'secondary';
  }

  confirmDeleteFrame(frame: FrameSchema, event: Event) {
    event.stopPropagation();
    this.confirmationService.confirm({
      message: `Delete frame <strong>${frame.name}</strong>? This cannot be undone.`,
      header: 'Delete Frame',
      icon: 'pi pi-exclamation-triangle',
      acceptButtonStyleClass: 'p-button-danger',
      accept: () => {
        this.vizService.deleteFrame(
          this.workspaceId(),
          frame.name,
          () => {
            this.messageService.add({ severity: 'success', summary: 'Deleted', detail: `Frame '${frame.name}' removed` });
            this.framesResult()?.refresh();
          },
        ).execute();
      },
    });
  }

  goBack() {
    this.router.navigate(['/workspaces']);
  }
}
