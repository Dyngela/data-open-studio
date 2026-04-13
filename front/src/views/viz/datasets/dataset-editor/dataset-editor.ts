import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule, ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { Button } from 'primeng/button';
import { InputText } from 'primeng/inputtext';
import { Textarea } from 'primeng/textarea';
import { Select } from 'primeng/select';
import { TableModule } from 'primeng/table';
import { Tag } from 'primeng/tag';
import { TabsModule } from 'primeng/tabs';
import { Toast } from 'primeng/toast';
import { ProgressSpinner } from 'primeng/progressspinner';
import { Dialog } from 'primeng/dialog';
import { MessageService } from 'primeng/api';

import { DatasetService } from '../../../../core/api/dataset.service';
import { MetadataService } from '../../../../core/api/metadata.service';
import { VizService } from '../../../../core/api/viz.service';
import {
  DatasetWithDetails,
  DatasetColumn,
  UpdateDatasetRequest,
  DatasetPreviewResult,
} from '../../../../core/api/dataset.type';
import { Workspace } from '../../../../core/api/viz.type';

@Component({
  selector: 'app-dataset-editor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    Button,
    InputText,
    Textarea,
    Select,
    TableModule,
    Tag,
    TabsModule,
    Toast,
    ProgressSpinner,
    Dialog,
  ],
  providers: [MessageService],
  templateUrl: './dataset-editor.html',
  styleUrl: './dataset-editor.css',
})
export class DatasetEditor implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private datasetService = inject(DatasetService);
  private metadataService = inject(MetadataService);
  private vizService = inject(VizService);
  private fb = inject(FormBuilder);
  private messageService = inject(MessageService);

  datasetId = signal<number>(0);
  dataset = signal<DatasetWithDetails | null>(null);
  previewResult = signal<DatasetPreviewResult | null>(null);
  activeTab = signal('schema');
  isDirty = signal(false);

  dbsResult = this.metadataService.getAllDb();
  databases = computed(() => this.dbsResult.data() ?? []);

  datasetResult = signal<ReturnType<typeof this.datasetService.getById> | null>(null);
  isLoading = computed(() => this.datasetResult()?.isLoading() ?? true);

  saveMutation = signal<ReturnType<typeof this.datasetService.update> | null>(null);
  isSaving = computed(() => this.saveMutation()?.isLoading() ?? false);

  refreshMutation = signal<ReturnType<typeof this.datasetService.refresh> | null>(null);
  isRefreshing = computed(() => this.refreshMutation()?.isLoading() ?? false);

  previewMutation = signal<ReturnType<typeof this.datasetService.preview> | null>(null);
  isPreviewing = computed(() => this.previewMutation()?.isLoading() ?? false);

  // Load-as-frame dialog
  showFrameDialog = signal(false);
  selectedWorkspaceId = signal<string | null>(null);
  workspacesResult = signal<ReturnType<typeof this.vizService.listWorkspaces> | null>(null);
  workspaces = computed<Workspace[]>(() => this.workspacesResult()?.data()?.workspaces ?? []);
  isLoadingWorkspaces = computed(() => this.workspacesResult()?.isLoading() ?? false);
  loadAsFrameMutation = signal<ReturnType<typeof this.datasetService.loadAsFrame> | null>(null);
  isLoadingFrame = computed(() => this.loadAsFrameMutation()?.isLoading() ?? false);

  frameName = computed(() => {
    const name = this.dataset()?.name ?? '';
    return name.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_|_$/g, '') || 'frame';
  });

  editForm = this.fb.group({
    name: ['', [Validators.required, Validators.minLength(2)]],
    description: [''],
    metadataDatabaseId: [null as number | null, Validators.required],
    query: ['', Validators.required],
  });

  ngOnInit() {
    this.dbsResult.refresh();
    const id = Number(this.route.snapshot.paramMap.get('id'));
    this.datasetId.set(id);
    this.loadDataset(id);
  }

  private loadDataset(id: number) {
    const result = this.datasetService.getById(id, (data) => {
      this.dataset.set(data);
      this.editForm.patchValue({
        name: data.name,
        description: data.description,
        metadataDatabaseId: data.metadataDatabaseId,
        query: data.query,
      });
      this.editForm.markAsPristine();
      this.isDirty.set(false);
    });
    this.datasetResult.set(result);
    result.refresh();

    this.editForm.valueChanges.subscribe(() => {
      this.isDirty.set(this.editForm.dirty);
    });
  }

  get databaseOptions() {
    return this.databases().map(db => ({
      label: `${db.host}/${db.databaseName} (${db.databaseType})`,
      value: db.id,
    }));
  }

  get columns(): DatasetColumn[] {
    return this.dataset()?.schema?.columns ?? [];
  }

  save() {
    if (this.editForm.invalid) return;
    const value = this.editForm.value;

    const mutation = this.datasetService.update(
      this.datasetId(),
      (updated) => {
        this.dataset.set(updated);
        this.editForm.markAsPristine();
        this.isDirty.set(false);
        this.messageService.add({ severity: 'success', summary: 'Saved', detail: 'Dataset updated' });
      },
      (err) => {
        this.messageService.add({ severity: 'error', summary: 'Error', detail: err?.error?.message || 'Failed to save' });
      }
    );

    this.saveMutation.set(mutation);
    mutation.execute({
      name: value.name!,
      description: value.description || undefined,
      metadataDatabaseId: value.metadataDatabaseId!,
      query: value.query!,
    } as UpdateDatasetRequest);
  }

  refreshSchema() {
    const mutation = this.datasetService.refresh(
      this.datasetId(),
      (updated) => {
        this.dataset.set(updated);
        this.messageService.add({
          severity: updated.status === 'error' ? 'warn' : 'success',
          summary: updated.status === 'error' ? 'Schema refresh failed' : 'Schema refreshed',
          detail: updated.status === 'error' ? updated.lastError : `${updated.schema.columns.length} columns detected`,
        });
      },
      (err) => {
        this.messageService.add({ severity: 'error', summary: 'Error', detail: err?.error?.message || 'Refresh failed' });
      }
    );
    this.refreshMutation.set(mutation);
    mutation.execute();
  }

  runPreview() {
    const mutation = this.datasetService.preview(
      this.datasetId(),
      (result) => {
        this.previewResult.set(result);
        this.activeTab.set('preview');
        this.messageService.add({ severity: 'info', summary: 'Preview loaded', detail: `${result.rowCount} rows` });
      },
      (err) => {
        this.messageService.add({ severity: 'error', summary: 'Query error', detail: err?.error?.message || 'Preview failed' });
      }
    );
    this.previewMutation.set(mutation);
    mutation.execute({ limit: 100 });
  }

  statusSeverity(status: string): 'success' | 'warn' | 'danger' | 'secondary' {
    switch (status) {
      case 'ready': return 'success';
      case 'error': return 'danger';
      default: return 'secondary';
    }
  }

  dataTypeSeverity(type: string): 'info' | 'secondary' | 'success' | 'warn' {
    switch (type) {
      case 'integer':
      case 'float': return 'info';
      case 'date':
      case 'datetime': return 'warn';
      case 'boolean': return 'success';
      default: return 'secondary';
    }
  }

  openFrameDialog() {
    this.selectedWorkspaceId.set(null);
    const result = this.vizService.listWorkspaces();
    this.workspacesResult.set(result);
    this.showFrameDialog.set(true);
  }

  confirmLoadAsFrame() {
    const workspaceId = this.selectedWorkspaceId();
    if (!workspaceId) return;

    const mutation = this.datasetService.loadAsFrame(
      this.datasetId(),
      (result) => {
        this.showFrameDialog.set(false);
        this.messageService.add({
          severity: 'success',
          summary: 'Frame loaded',
          detail: `Frame "${result.frame_name}" is ready in the workspace`,
        });
      },
      (err) => {
        this.messageService.add({
          severity: 'error',
          summary: 'Load failed',
          detail: err?.error?.message || 'Failed to load frame',
        });
      }
    );
    this.loadAsFrameMutation.set(mutation);
    mutation.execute({ workspaceId });
  }

  goBack() {
    this.router.navigate(['/datasets']);
  }
}
