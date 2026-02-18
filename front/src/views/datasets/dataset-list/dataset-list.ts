import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { FormsModule, ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { Button } from 'primeng/button';
import { TableModule } from 'primeng/table';
import { Tag } from 'primeng/tag';
import { Dialog } from 'primeng/dialog';
import { InputText } from 'primeng/inputtext';
import { Textarea } from 'primeng/textarea';
import { Select } from 'primeng/select';
import { ConfirmDialog } from 'primeng/confirmdialog';
import { ConfirmationService, MessageService } from 'primeng/api';
import { Toast } from 'primeng/toast';
import { Tooltip } from 'primeng/tooltip';

import { DatasetService } from '../../../core/api/dataset.service';
import { MetadataService } from '../../../core/api/metadata.service';
import { Dataset, CreateDatasetRequest } from '../../../core/api/dataset.type';

@Component({
  selector: 'app-dataset-list',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    Button,
    TableModule,
    Tag,
    Dialog,
    InputText,
    Textarea,
    Select,
    ConfirmDialog,
    Toast,
    Tooltip,
  ],
  providers: [ConfirmationService, MessageService],
  templateUrl: './dataset-list.html',
  styleUrl: './dataset-list.css',
})
export class DatasetList implements OnInit {
  private datasetService = inject(DatasetService);
  private metadataService = inject(MetadataService);
  private router = inject(Router);
  private fb = inject(FormBuilder);
  private confirmationService = inject(ConfirmationService);
  private messageService = inject(MessageService);

  // Data
  datasetsResult = this.datasetService.getAll();
  datasets = computed(() => this.datasetsResult.data() ?? []);
  isLoading = this.datasetsResult.isLoading;

  dbsResult = this.metadataService.getAllDb();
  databases = computed(() => this.dbsResult.data() ?? []);

  // Dialog state
  showCreateDialog = signal(false);
  createMutation = signal<ReturnType<typeof this.datasetService.create> | null>(null);
  isCreating = computed(() => this.createMutation()?.isLoading() ?? false);

  createForm = this.fb.group({
    name: ['', [Validators.required, Validators.minLength(2)]],
    description: [''],
    metadataDatabaseId: [null as number | null, Validators.required],
    query: ['SELECT * FROM ', Validators.required],
  });

  ngOnInit() {
    this.datasetsResult.refresh();
    this.dbsResult.refresh();
  }

  get databaseOptions() {
    return this.databases().map(db => ({
      label: `${db.host}/${db.databaseName} (${db.databaseType})`,
      value: db.id,
    }));
  }

  openCreateDialog() {
    this.createForm.reset({ query: 'SELECT * FROM ' });
    this.showCreateDialog.set(true);
  }

  submitCreate() {
    if (this.createForm.invalid) return;

    const mutation = this.datasetService.create(
      (created) => {
        this.messageService.add({ severity: 'success', summary: 'Dataset created', detail: created.name });
        this.showCreateDialog.set(false);
        this.datasetsResult.refresh();
        this.router.navigate(['/datasets', created.id]);
      },
      (err) => {
        this.messageService.add({ severity: 'error', summary: 'Error', detail: err?.error?.message || 'Failed to create dataset' });
      }
    );

    this.createMutation.set(mutation);

    const value = this.createForm.value;
    mutation.execute({
      name: value.name!,
      description: value.description || '',
      metadataDatabaseId: value.metadataDatabaseId!,
      query: value.query!,
    } as CreateDatasetRequest);
  }

  openDataset(dataset: Dataset) {
    this.router.navigate(['/datasets', dataset.id]);
  }

  confirmDelete(event: Event, dataset: Dataset) {
    this.confirmationService.confirm({
      target: event.target as EventTarget,
      message: `Delete "${dataset.name}"? This cannot be undone.`,
      icon: 'pi pi-exclamation-triangle',
      accept: () => {
        const mutation = this.datasetService.deleteDataset(
          dataset.id,
          () => {
            this.messageService.add({ severity: 'success', summary: 'Deleted', detail: dataset.name });
            this.datasetsResult.refresh();
          },
          () => {
            this.messageService.add({ severity: 'error', summary: 'Error', detail: 'Failed to delete dataset' });
          }
        );
        mutation.execute();
      },
    });
  }

  statusSeverity(status: string): 'success' | 'warn' | 'danger' | 'secondary' {
    switch (status) {
      case 'ready': return 'success';
      case 'error': return 'danger';
      default: return 'secondary';
    }
  }

  getDatabaseName(id: number): string {
    const db = this.databases().find(d => d.id === id);
    return db ? `${db.host}/${db.databaseName}` : `DB #${id}`;
  }
}
