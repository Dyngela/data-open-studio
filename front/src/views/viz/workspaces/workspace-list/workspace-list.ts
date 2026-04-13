import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { FormsModule, ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { Button } from 'primeng/button';
import { TableModule } from 'primeng/table';
import { Dialog } from 'primeng/dialog';
import { InputText } from 'primeng/inputtext';
import { ConfirmDialog } from 'primeng/confirmdialog';
import { Toast } from 'primeng/toast';
import { Tooltip } from 'primeng/tooltip';
import { ConfirmationService, MessageService } from 'primeng/api';

import { VizService } from '../../../../core/api/viz.service';
import { Workspace } from '../../../../core/api/viz.type';

@Component({
  selector: 'app-workspace-list',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    Button,
    TableModule,
    Dialog,
    InputText,
    ConfirmDialog,
    Toast,
    Tooltip,
  ],
  providers: [ConfirmationService, MessageService],
  templateUrl: './workspace-list.html',
})
export class WorkspaceList implements OnInit {
  private vizService = inject(VizService);
  private router = inject(Router);
  private fb = inject(FormBuilder);
  private confirmationService = inject(ConfirmationService);
  private messageService = inject(MessageService);

  workspacesResult = this.vizService.listWorkspaces();
  workspaces = computed(() => this.workspacesResult.data()?.workspaces ?? []);
  isLoading = this.workspacesResult.isLoading;

  // Create dialog
  showCreateDialog = signal(false);
  createMutation = signal<ReturnType<typeof this.vizService.createWorkspace> | null>(null);
  isCreating = computed(() => this.createMutation()?.isLoading() ?? false);

  createForm = this.fb.group({
    name: ['', [Validators.required, Validators.minLength(2)]],
  });

  // Rename dialog
  showRenameDialog = signal(false);
  renamingWorkspace = signal<Workspace | null>(null);
  renameMutation = signal<ReturnType<typeof this.vizService.updateWorkspace> | null>(null);
  isRenaming = computed(() => this.renameMutation()?.isLoading() ?? false);

  renameForm = this.fb.group({
    name: ['', [Validators.required, Validators.minLength(2)]],
  });

  ngOnInit() {
    this.workspacesResult.refresh();
  }

  openCreateDialog() {
    this.createForm.reset();
    this.showCreateDialog.set(true);
  }

  submitCreate() {
    if (this.createForm.invalid) return;

    const mutation = this.vizService.createWorkspace(
      (created) => {
        this.messageService.add({ severity: 'success', summary: 'Workspace created', detail: created.name });
        this.showCreateDialog.set(false);
        this.workspacesResult.refresh();
      },
      (err) => {
        this.messageService.add({ severity: 'error', summary: 'Error', detail: err.message || 'Failed to create workspace' });
      }
    );

    this.createMutation.set(mutation);
    mutation.execute({ name: this.createForm.value.name! });
  }

  openWorkspace(ws: Workspace) {
    this.router.navigate(['/workspaces', ws.id]);
  }

  openRenameDialog(ws: Workspace) {
    this.renamingWorkspace.set(ws);
    this.renameForm.setValue({ name: ws.name });
    this.showRenameDialog.set(true);
  }

  submitRename() {
    const ws = this.renamingWorkspace();
    if (!ws || this.renameForm.invalid) return;

    const mutation = this.vizService.updateWorkspace(
      ws.id,
      (updated) => {
        this.messageService.add({ severity: 'success', summary: 'Renamed', detail: updated.name });
        this.showRenameDialog.set(false);
        this.workspacesResult.refresh();
      },
      (err) => {
        this.messageService.add({ severity: 'error', summary: 'Error', detail: err.message || 'Failed to rename workspace' });
      }
    );

    this.renameMutation.set(mutation);
    mutation.execute({ name: this.renameForm.value.name! });
  }

  confirmDelete(event: Event, ws: Workspace) {
    this.confirmationService.confirm({
      target: event.target as EventTarget,
      message: `Delete "${ws.name}"? All sources inside will be removed.`,
      icon: 'pi pi-exclamation-triangle',
      accept: () => {
        const mutation = this.vizService.deleteWorkspace(
          ws.id,
          () => {
            this.messageService.add({ severity: 'success', summary: 'Deleted', detail: ws.name });
            this.workspacesResult.refresh();
          },
          (err) => {
            this.messageService.add({ severity: 'error', summary: 'Error', detail: err.message || 'Failed to delete workspace' });
          }
        );
        mutation.execute();
      },
    });
  }
}
