import { Component, inject, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Button } from 'primeng/button';
import { Dialog } from 'primeng/dialog';
import { InputText } from 'primeng/inputtext';
import { InputNumber } from 'primeng/inputnumber';
import { Select } from 'primeng/select';
import { TableModule } from 'primeng/table';
import { Tag } from 'primeng/tag';
import { Tooltip } from 'primeng/tooltip';
import { TabsModule } from 'primeng/tabs';
import { ConfirmDialog } from 'primeng/confirmdialog';
import { ConfirmationService, MenuItem, MessageService } from 'primeng/api';
import { Toast } from 'primeng/toast';
import { Chip } from 'primeng/chip';

import { TriggerService } from '../../../core/api/trigger.service';
import { JobService } from '../../../core/api/job.service';
import { MetadataService } from '../../../core/api/metadata.service';
import { DbMetadata } from '../../../core/api/metadata.type';
import {
  Trigger,
  TriggerWithDetails,
  TriggerType,
  TriggerStatus,
  CreateTriggerRequest,
  UpdateTriggerRequest,
  TriggerConfig,
  DatabaseTable,
  DatabaseColumn,
  TriggerRule,
  TriggerJobLink,
  TriggerExecution,
  RuleConditions,
  ConditionOperator,
} from '../../../core/api/trigger.type';
import { Job } from '../../../core/api/job.type';

interface TriggerTypeOption {
  label: string;
  value: TriggerType;
  icon: string;
  description: string;
}

@Component({
  selector: 'app-triggers',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    Button,
    Dialog,
    InputText,
    InputNumber,
    Select,
    TableModule,
    Tag,
    Tooltip,
    TabsModule,
    ConfirmDialog,
    Toast,
    Chip,
  ],
  providers: [ConfirmationService, MessageService],
  templateUrl: './triggers.html',
  styleUrl: './triggers.css',
})
export class Triggers {
  private triggerService = inject(TriggerService);
  private jobService = inject(JobService);
  private metadataService = inject(MetadataService);
  private fb = inject(FormBuilder);
  private confirmationService = inject(ConfirmationService);
  private messageService = inject(MessageService);

  // Data
  triggersResult = this.triggerService.getAll();
  triggers = computed(() => this.triggersResult.data() ?? []);
  isLoading = this.triggersResult.isLoading;

  jobsResult = this.jobService.getAll();
  availableJobs = computed(() => this.jobsResult.data() ?? []);

  // Metadata DB connections
  dbMetadataResult = this.metadataService.getAllDb();
  dbConnections = computed(() => this.dbMetadataResult.data() ?? []);
  selectedDbConnection = signal<DbMetadata | null>(null);

  // Selected trigger for details view
  selectedTrigger = signal<TriggerWithDetails | null>(null);
  selectedTriggerResult = signal<ReturnType<typeof this.triggerService.getById> | null>(null);
  triggerExecutions = signal<TriggerExecution[]>([]);

  // Create/Edit modal
  showCreateModal = signal(false);
  isEditing = signal(false);
  editingTriggerId = signal<number | null>(null);
  currentStep = signal(0);
  isSubmitting = signal(false);

  // Database introspection
  availableTables = signal<DatabaseTable[]>([]);
  availableColumns = signal<DatabaseColumn[]>([]);
  isLoadingTables = signal(false);
  isLoadingColumns = signal(false);

  // Rule modal
  showRuleModal = signal(false);
  editingRule = signal<TriggerRule | null>(null);
  ruleForm: FormGroup;

  // Link job modal
  showLinkJobModal = signal(false);
  selectedJobToLink = signal<Job | null>(null);

  // Wizard job selection (step 2)
  selectedJobsForLink = signal<Job[]>([]);

  // Trigger type options
  triggerTypes: TriggerTypeOption[] = [
    {
      label: 'Base de données',
      value: 'database',
      icon: 'pi pi-database',
      description: 'Surveille une table et déclenche un job quand de nouvelles lignes apparaissent',
    },
    {
      label: 'Email',
      value: 'email',
      icon: 'pi pi-envelope',
      description: 'Surveille une boîte mail et déclenche un job sur réception d\'un email',
    },
    {
      label: 'Webhook',
      value: 'webhook',
      icon: 'pi pi-globe',
      description: 'Expose un endpoint HTTP pour déclencher un job',
    },
  ];

  watermarkTypes = [
    { label: 'Entier (ID)', value: 'int' },
    { label: 'Timestamp', value: 'timestamp' },
    { label: 'UUID', value: 'uuid' },
  ];

  conditionOperators = [
    { label: 'Égal à', value: 'eq' },
    { label: 'Différent de', value: 'neq' },
    { label: 'Contient', value: 'contains' },
    { label: 'Commence par', value: 'startsWith' },
    { label: 'Termine par', value: 'endsWith' },
    { label: 'Supérieur à', value: 'gt' },
    { label: 'Inférieur à', value: 'lt' },
    { label: 'Correspond à (regex)', value: 'regex' },
    { label: 'Existe', value: 'exists' },
    { label: 'N\'existe pas', value: 'notExists' },
  ];

  wizardSteps: MenuItem[] = [
    { label: 'Type' },
    { label: 'Configuration' },
    { label: 'Jobs' },
    { label: 'Résumé' },
  ];

  // Forms
  triggerForm: FormGroup;
  tableForm: FormGroup;

  constructor() {
    // Main trigger form
    this.triggerForm = this.fb.group({
      name: ['', [Validators.required, Validators.maxLength(100)]],
      description: ['', [Validators.maxLength(500)]],
      type: ['database' as TriggerType, Validators.required],
      pollingInterval: [60, [Validators.required, Validators.min(10), Validators.max(86400)]],
    });

    // Table selection form
    this.tableForm = this.fb.group({
      tableName: ['', Validators.required],
      watermarkColumn: ['', Validators.required],
      watermarkType: ['int', Validators.required],
      batchSize: [100, [Validators.min(1), Validators.max(1000)]],
    });

    // Rule form
    this.ruleForm = this.fb.group({
      name: [''],
      field: ['', Validators.required],
      operator: ['eq' as ConditionOperator, Validators.required],
      value: [''],
    });
  }

  // Status badge helpers
  getStatusSeverity(status: TriggerStatus): 'success' | 'secondary' | 'info' | 'warn' | 'danger' | 'contrast' | undefined {
    const severities: Record<TriggerStatus, 'success' | 'secondary' | 'info' | 'warn' | 'danger'> = {
      active: 'success',
      paused: 'secondary',
      error: 'danger',
      disabled: 'warn',
    };
    return severities[status];
  }

  getStatusLabel(status: TriggerStatus): string {
    const labels: Record<TriggerStatus, string> = {
      active: 'Actif',
      paused: 'En pause',
      error: 'Erreur',
      disabled: 'Désactivé',
    };
    return labels[status];
  }

  getTypeIcon(type: TriggerType): string {
    const icons: Record<TriggerType, string> = {
      database: 'pi pi-database',
      email: 'pi pi-envelope',
      webhook: 'pi pi-globe',
    };
    return icons[type];
  }

  getTypeLabel(type: TriggerType): string {
    const labels: Record<TriggerType, string> = {
      database: 'Base de données',
      email: 'Email',
      webhook: 'Webhook',
    };
    return labels[type];
  }

  // Open trigger details
  viewTrigger(trigger: Trigger) {
    const result = this.triggerService.getById(trigger.id);
    this.selectedTriggerResult.set(result);

    // Watch for data changes
    effect(() => {
      const data = result.data();
      if (data) {
        this.selectedTrigger.set(data);
        this.loadExecutions(trigger.id);
      }
    });
  }

  loadExecutions(triggerId: number) {
    const result = this.triggerService.getExecutions(triggerId, 20);
    effect(() => {
      const data = result.data();
      if (data) {
        this.triggerExecutions.set(data);
      }
    }, { allowSignalWrites: true });
  }

  closeTriggerDetails() {
    this.selectedTrigger.set(null);
    this.selectedTriggerResult.set(null);
    this.triggerExecutions.set([]);
  }

  // Create/Edit modal
  openCreateModal() {
    this.isEditing.set(false);
    this.editingTriggerId.set(null);
    this.currentStep.set(0);
    this.resetForms();
    this.showCreateModal.set(true);
  }

  openEditModal(trigger: TriggerWithDetails) {
    this.isEditing.set(true);
    this.editingTriggerId.set(trigger.id);
    this.currentStep.set(0);

    // Populate forms with existing data
    this.triggerForm.patchValue({
      name: trigger.name,
      description: trigger.description,
      type: trigger.type,
      pollingInterval: trigger.pollingInterval,
    });

    if (trigger.type === 'database' && trigger.config.database) {
      const dbConfig = trigger.config.database;
      // Pre-select the metadata DB connection if one was used
      if (dbConfig.metadataDatabaseId) {
        const match = this.dbConnections().find(c => c.id === dbConfig.metadataDatabaseId);
        if (match) {
          this.selectedDbConnection.set(match);
          this.onDbConnectionSelect(match);
        }
      }
      this.tableForm.patchValue({
        tableName: dbConfig.tableName,
        watermarkColumn: dbConfig.watermarkColumn,
        watermarkType: dbConfig.watermarkType,
        batchSize: dbConfig.batchSize || 100,
      });
    }

    this.showCreateModal.set(true);
  }

  closeCreateModal() {
    this.showCreateModal.set(false);
    this.resetForms();
  }

  resetForms() {
    this.triggerForm.reset({
      name: '',
      description: '',
      type: 'database',
      pollingInterval: 60,
    });
    this.tableForm.reset({
      tableName: '',
      watermarkColumn: '',
      watermarkType: 'int',
      batchSize: 100,
    });
    this.selectedDbConnection.set(null);
    this.availableTables.set([]);
    this.availableColumns.set([]);
    this.selectedJobsForLink.set([]);
  }

  // Wizard navigation
  nextStep() {
    if (this.currentStep() < 3) {
      this.currentStep.update(s => s + 1);
    }
  }

  prevStep() {
    if (this.currentStep() > 0) {
      this.currentStep.update(s => s - 1);
    }
  }

  canProceedToNextStep(): boolean {
    const step = this.currentStep();
    if (step === 0) {
      return this.triggerForm.valid;
    }
    if (step === 1) {
      const type = this.triggerForm.get('type')?.value;
      if (type === 'database') {
        return !!this.selectedDbConnection() && this.tableForm.valid;
      }
      return true;
    }
    if (step === 2) {
      // Jobs step is optional - always can proceed
      return true;
    }
    return true;
  }

  // Database connection selection
  onDbConnectionSelect(db: DbMetadata) {
    this.selectedDbConnection.set(db);
    this.availableTables.set([]);
    this.availableColumns.set([]);
    this.tableForm.patchValue({ tableName: '', watermarkColumn: '' });
    this.loadTables();
  }

  getDbConnectionLabel(db: DbMetadata): string {
    return `${db.databaseName} (${db.host}:${db.port})`;
  }

  loadTables() {
    const db = this.selectedDbConnection();
    if (!db) return;

    this.isLoadingTables.set(true);

    const mutation = this.triggerService.getTables(
      (result) => {
        this.isLoadingTables.set(false);
        this.availableTables.set(result.tables || []);
      },
      () => {
        this.isLoadingTables.set(false);
      }
    );
    mutation.execute({ metadataDatabaseId: db.id });
  }

  onTableSelect(event: any) {
    const tableName = event.value;
    if (!tableName) return;

    const db = this.selectedDbConnection();
    if (!db) return;

    this.isLoadingColumns.set(true);

    const mutation = this.triggerService.getColumns(
      (result) => {
        this.isLoadingColumns.set(false);
        this.availableColumns.set(result.columns || []);
      },
      () => {
        this.isLoadingColumns.set(false);
      }
    );
    mutation.execute({ metadataDatabaseId: db.id, tableName });
  }

  // Create/Update trigger
  saveTrigger() {
    if (!this.canProceedToNextStep()) return;

    this.isSubmitting.set(true);

    const triggerData = this.triggerForm.value;
    const config = this.buildTriggerConfig();

    if (this.isEditing() && this.editingTriggerId()) {
      const request: UpdateTriggerRequest = {
        name: triggerData.name,
        description: triggerData.description,
        pollingInterval: triggerData.pollingInterval,
        config,
      };

      const mutation = this.triggerService.update(
        this.editingTriggerId()!,
        () => {
          this.isSubmitting.set(false);
          this.closeCreateModal();
          this.triggersResult.refresh();
          this.messageService.add({
            severity: 'success',
            summary: 'Succès',
            detail: 'Trigger mis à jour',
          });
        },
        () => {
          this.isSubmitting.set(false);
          this.messageService.add({
            severity: 'error',
            summary: 'Erreur',
            detail: 'Impossible de mettre à jour le trigger',
          });
        }
      );
      mutation.execute(request);
    } else {
      const request: CreateTriggerRequest = {
        name: triggerData.name,
        description: triggerData.description,
        type: triggerData.type,
        pollingInterval: triggerData.pollingInterval,
        config,
      };

      const jobsToLink = this.selectedJobsForLink();
      const mutation = this.triggerService.create(
        (created) => {
          // Link selected jobs after creation
          if (jobsToLink.length > 0) {
            let linked = 0;
            for (const job of jobsToLink) {
              const linkMutation = this.triggerService.linkJob(
                created.id,
                () => {
                  linked++;
                  if (linked === jobsToLink.length) {
                    this.isSubmitting.set(false);
                    this.closeCreateModal();
                    this.triggersResult.refresh();
                    this.messageService.add({
                      severity: 'success',
                      summary: 'Succès',
                      detail: `Trigger créé avec ${linked} job(s) lié(s)`,
                    });
                  }
                },
                () => {
                  linked++;
                  if (linked === jobsToLink.length) {
                    this.isSubmitting.set(false);
                    this.closeCreateModal();
                    this.triggersResult.refresh();
                    this.messageService.add({
                      severity: 'warn',
                      summary: 'Attention',
                      detail: 'Trigger créé mais certains jobs n\'ont pas pu être liés',
                    });
                  }
                }
              );
              linkMutation.execute({ jobId: job.id });
            }
          } else {
            this.isSubmitting.set(false);
            this.closeCreateModal();
            this.triggersResult.refresh();
            this.messageService.add({
              severity: 'success',
              summary: 'Succès',
              detail: 'Trigger créé',
            });
          }
        },
        () => {
          this.isSubmitting.set(false);
          this.messageService.add({
            severity: 'error',
            summary: 'Erreur',
            detail: 'Impossible de créer le trigger',
          });
        }
      );
      mutation.execute(request);
    }
  }

  buildTriggerConfig(): TriggerConfig {
    const type = this.triggerForm.get('type')?.value as TriggerType;

    if (type === 'database') {
      const db = this.selectedDbConnection();
      const table = this.tableForm.value;

      return {
        database: {
          metadataDatabaseId: db?.id,
          tableName: table.tableName,
          watermarkColumn: table.watermarkColumn,
          watermarkType: table.watermarkType,
          batchSize: table.batchSize,
        },
      };
    }

    // For other types, return empty config for now
    return {};
  }

  // Activate/Pause trigger
  activateTrigger(trigger: Trigger) {
    const mutation = this.triggerService.activate(
      trigger.id,
      () => {
        this.triggersResult.refresh();
        if (this.selectedTrigger()?.id === trigger.id) {
          this.refreshSelectedTrigger();
        }
        this.messageService.add({
          severity: 'success',
          summary: 'Succès',
          detail: 'Trigger activé',
        });
      },
      () => {
        this.messageService.add({
          severity: 'error',
          summary: 'Erreur',
          detail: 'Impossible d\'activer le trigger',
        });
      }
    );
    mutation.execute();
  }

  pauseTrigger(trigger: Trigger) {
    const mutation = this.triggerService.pause(
      trigger.id,
      () => {
        this.triggersResult.refresh();
        if (this.selectedTrigger()?.id === trigger.id) {
          this.refreshSelectedTrigger();
        }
        this.messageService.add({
          severity: 'info',
          summary: 'Info',
          detail: 'Trigger mis en pause',
        });
      },
      () => {
        this.messageService.add({
          severity: 'error',
          summary: 'Erreur',
          detail: 'Impossible de mettre en pause le trigger',
        });
      }
    );
    mutation.execute();
  }

  // Delete trigger
  deleteTrigger(trigger: Trigger) {
    this.confirmationService.confirm({
      message: `Êtes-vous sûr de vouloir supprimer le trigger "${trigger.name}" ?`,
      header: 'Confirmation',
      icon: 'pi pi-exclamation-triangle',
      accept: () => {
        const mutation = this.triggerService.deleteTrigger(
          trigger.id,
          () => {
            this.triggersResult.refresh();
            if (this.selectedTrigger()?.id === trigger.id) {
              this.closeTriggerDetails();
            }
            this.messageService.add({
              severity: 'success',
              summary: 'Succès',
              detail: 'Trigger supprimé',
            });
          }
        );
        mutation.execute();
      },
    });
  }

  refreshSelectedTrigger() {
    const trigger = this.selectedTrigger();
    if (trigger) {
      this.viewTrigger(trigger);
    }
  }

  // Rule management
  openAddRuleModal() {
    this.editingRule.set(null);
    this.ruleForm.reset({
      name: '',
      field: '',
      operator: 'eq',
      value: '',
    });
    this.showRuleModal.set(true);
  }

  openEditRuleModal(rule: TriggerRule) {
    this.editingRule.set(rule);
    const condition = rule.conditions.all?.[0] || rule.conditions.any?.[0];
    this.ruleForm.patchValue({
      name: rule.name,
      field: condition?.field || '',
      operator: condition?.operator || 'eq',
      value: condition?.value || '',
    });
    this.showRuleModal.set(true);
  }

  closeRuleModal() {
    this.showRuleModal.set(false);
    this.editingRule.set(null);
  }

  saveRule() {
    if (this.ruleForm.invalid || !this.selectedTrigger()) return;

    const trigger = this.selectedTrigger()!;
    const formValue = this.ruleForm.value;

    const conditions: RuleConditions = {
      all: [{
        field: formValue.field,
        operator: formValue.operator,
        value: formValue.value,
      }],
    };

    if (this.editingRule()) {
      const mutation = this.triggerService.updateRule(
        trigger.id,
        this.editingRule()!.id,
        () => {
          this.closeRuleModal();
          this.refreshSelectedTrigger();
          this.messageService.add({
            severity: 'success',
            summary: 'Succès',
            detail: 'Règle mise à jour',
          });
        }
      );
      mutation.execute({ name: formValue.name, conditions });
    } else {
      const mutation = this.triggerService.addRule(
        trigger.id,
        () => {
          this.closeRuleModal();
          this.refreshSelectedTrigger();
          this.messageService.add({
            severity: 'success',
            summary: 'Succès',
            detail: 'Règle ajoutée',
          });
        }
      );
      mutation.execute({ name: formValue.name, conditions });
    }
  }

  deleteRule(rule: TriggerRule) {
    const trigger = this.selectedTrigger();
    if (!trigger) return;

    this.confirmationService.confirm({
      message: 'Supprimer cette règle ?',
      header: 'Confirmation',
      icon: 'pi pi-exclamation-triangle',
      accept: () => {
        const mutation = this.triggerService.deleteRule(
          trigger.id,
          rule.id,
          () => {
            this.refreshSelectedTrigger();
            this.messageService.add({
              severity: 'success',
              summary: 'Succès',
              detail: 'Règle supprimée',
            });
          }
        );
        mutation.execute();
      },
    });
  }

  // Job linking
  openLinkJobModal() {
    this.selectedJobToLink.set(null);
    this.showLinkJobModal.set(true);
  }

  closeLinkJobModal() {
    this.showLinkJobModal.set(false);
    this.selectedJobToLink.set(null);
  }

  linkJob() {
    const trigger = this.selectedTrigger();
    const job = this.selectedJobToLink();
    if (!trigger || !job) return;

    const mutation = this.triggerService.linkJob(
      trigger.id,
      () => {
        this.closeLinkJobModal();
        this.refreshSelectedTrigger();
        this.messageService.add({
          severity: 'success',
          summary: 'Succès',
          detail: 'Job lié au trigger',
        });
      }
    );
    mutation.execute({ jobId: job.id });
  }

  unlinkJob(jobLink: TriggerJobLink) {
    const trigger = this.selectedTrigger();
    if (!trigger) return;

    this.confirmationService.confirm({
      message: `Retirer le job "${jobLink.jobName}" de ce trigger ?`,
      header: 'Confirmation',
      icon: 'pi pi-exclamation-triangle',
      accept: () => {
        const mutation = this.triggerService.unlinkJob(
          trigger.id,
          jobLink.jobId,
          () => {
            this.refreshSelectedTrigger();
            this.messageService.add({
              severity: 'success',
              summary: 'Succès',
              detail: 'Job retiré',
            });
          }
        );
        mutation.execute();
      },
    });
  }

  // Wizard job selection helpers
  toggleJobForLink(job: Job) {
    const current = this.selectedJobsForLink();
    const exists = current.find(j => j.id === job.id);
    if (exists) {
      this.selectedJobsForLink.set(current.filter(j => j.id !== job.id));
    } else {
      this.selectedJobsForLink.set([...current, job]);
    }
  }

  isJobSelectedForLink(jobId: number): boolean {
    return this.selectedJobsForLink().some(j => j.id === jobId);
  }

  // Helpers
  getLinkedJobIds(): number[] {
    return this.selectedTrigger()?.jobs.map(j => j.jobId) || [];
  }

  getAvailableJobsToLink(): Job[] {
    const linkedIds = this.getLinkedJobIds();
    return this.availableJobs().filter(j => !linkedIds.includes(j.id));
  }

  getDbConnectionName(metadataId: number): string {
    const db = this.dbConnections().find(c => c.id === metadataId);
    return db ? `${db.databaseName} (${db.host}:${db.port})` : `Connexion #${metadataId}`;
  }

  formatDate(dateStr: string | undefined): string {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleString('fr-FR');
  }

  getExecutionStatusSeverity(status: string): 'success' | 'secondary' | 'info' | 'warn' | 'danger' | 'contrast' | undefined {
    const severities: Record<string, 'success' | 'secondary' | 'info' | 'warn' | 'danger'> = {
      completed: 'success',
      running: 'info',
      failed: 'danger',
      no_events: 'secondary',
    };
    return severities[status] || 'secondary';
  }

  getExecutionStatusLabel(status: string): string {
    const labels: Record<string, string> = {
      completed: 'Terminé',
      running: 'En cours',
      failed: 'Échec',
      no_events: 'Aucun événement',
    };
    return labels[status] || status;
  }
}
