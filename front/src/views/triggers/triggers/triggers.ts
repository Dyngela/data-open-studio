import { Component, inject, signal, computed } from '@angular/core';
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
import { AutoComplete } from 'primeng/autocomplete';

import { TriggerService } from '../../../core/api/trigger.service';
import { SqlService } from '../../../core/api/sql.service';
import { JobService } from '../../../core/api/job.service';
import { UserService } from '../../../core/api/user.service';
import { MetadataService } from '../../../core/api/metadata.service';
import { DbMetadata, EmailMetadata } from '../../../core/api/metadata.type';
import {
  Trigger,
  TriggerWithDetails,
  TriggerType,
  TriggerStatus,
  CreateTriggerRequest,
  UpdateTriggerRequest,
  TriggerConfig,
  TriggerRule,
  TriggerJobLink,
  TriggerExecution,
  RuleConditions,
  ConditionOperator,
  CronMode,
  IntervalUnit,
  ScheduleFrequency,
} from '../../../core/api/trigger.type';
import { DatabaseTable, DatabaseColumn } from '../../../core/api/sql.type';
import { Job, NotificationContact, User } from '../../../core/api/job.type';

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
    AutoComplete,
  ],
  providers: [ConfirmationService, MessageService],
  templateUrl: './triggers.html',
  styleUrl: './triggers.css',
})
export class Triggers {
  private triggerService = inject(TriggerService);
  private sqlService = inject(SqlService);
  private jobService = inject(JobService);
  private userService = inject(UserService);
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

  // Metadata Email connections
  emailMetadataResult = this.metadataService.getAllEmail();
  emailConnections = computed(() => this.emailMetadataResult.data() ?? []);
  selectedEmailConnection = signal<EmailMetadata | null>(null);

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

  // Notification contacts
  notificationContacts = signal<Map<number, NotificationContact[]>>(new Map());
  filteredUsers = signal<User[]>([]);

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
    {
      label: 'Cron',
      value: 'cron',
      icon: 'pi pi-clock',
      description: 'Déclenche un job selon un intervalle ou un horaire planifié',
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

  // Cron options
  cronModes = [
    { label: 'Intervalle', value: 'interval', description: 'Exécuter toutes les X minutes/heures/jours' },
    { label: 'Planifié', value: 'schedule', description: 'Exécuter à un horaire précis' },
  ];

  intervalUnits = [
    { label: 'Minutes', value: 'minutes' },
    { label: 'Heures', value: 'hours' },
    { label: 'Jours', value: 'days' },
  ];

  scheduleFrequencies = [
    { label: 'Tous les jours', value: 'daily' },
    { label: 'Toutes les semaines', value: 'weekly' },
    { label: 'Tous les mois', value: 'monthly' },
  ];

  daysOfWeek = [
    { label: 'Dimanche', value: 0 },
    { label: 'Lundi', value: 1 },
    { label: 'Mardi', value: 2 },
    { label: 'Mercredi', value: 3 },
    { label: 'Jeudi', value: 4 },
    { label: 'Vendredi', value: 5 },
    { label: 'Samedi', value: 6 },
  ];

  // Forms
  triggerForm: FormGroup;
  tableForm: FormGroup;
  emailForm: FormGroup;
  cronForm: FormGroup;

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

    // Email form
    this.emailForm = this.fb.group({
      folder: ['INBOX'],
      fromAddress: [''],
      toAddress: [''],
      subjectPattern: [''],
      markAsRead: [false],
    });

    // Cron form
    this.cronForm = this.fb.group({
      mode: ['interval' as CronMode, Validators.required],
      intervalValue: [30, [Validators.required, Validators.min(1)]],
      intervalUnit: ['minutes' as IntervalUnit, Validators.required],
      scheduleFrequency: ['daily' as ScheduleFrequency],
      scheduleTime: ['09:00'],
      scheduleDayOfWeek: [1], // Monday
      scheduleDayOfMonth: [1],
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
      cron: 'pi pi-clock',
    };
    return icons[type];
  }

  getTypeLabel(type: TriggerType): string {
    const labels: Record<TriggerType, string> = {
      database: 'Base de données',
      email: 'Email',
      webhook: 'Webhook',
      cron: 'Cron',
    };
    return labels[type];
  }

  // Open trigger details
  viewTrigger(trigger: Trigger) {
    const result = this.triggerService.getById(trigger.id, (data) => {
      this.selectedTrigger.set(data);
      this.loadExecutions(trigger.id);
      this.loadNotificationContacts();
    });
    this.selectedTriggerResult.set(result);
  }

  loadExecutions(triggerId: number) {
    this.triggerService.getExecutions(triggerId, 20, (data) => {
      this.triggerExecutions.set(data);
    });
  }

  closeTriggerDetails() {
    this.selectedTrigger.set(null);
    this.selectedTriggerResult.set(null);
    this.triggerExecutions.set([]);
    this.notificationContacts.set(new Map());
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

    if (trigger.type === 'email' && trigger.config.email) {
      const emailConfig = trigger.config.email;
      if (emailConfig.metadataEmailId) {
        const match = this.emailConnections().find(c => c.id === emailConfig.metadataEmailId);
        if (match) {
          this.selectedEmailConnection.set(match);
        }
      }
      this.emailForm.patchValue({
        folder: emailConfig.folder || 'INBOX',
        fromAddress: emailConfig.fromAddress || '',
        toAddress: emailConfig.toAddress || '',
        subjectPattern: emailConfig.subjectPattern || '',
        markAsRead: emailConfig.markAsRead || false,
      });
    }

    if (trigger.type === 'cron' && trigger.config.cron) {
      const cronConfig = trigger.config.cron;
      this.cronForm.patchValue({
        mode: cronConfig.mode || 'interval',
        intervalValue: cronConfig.intervalValue || 30,
        intervalUnit: cronConfig.intervalUnit || 'minutes',
        scheduleFrequency: cronConfig.scheduleFrequency || 'daily',
        scheduleTime: cronConfig.scheduleTime || '09:00',
        scheduleDayOfWeek: cronConfig.scheduleDayOfWeek ?? 1,
        scheduleDayOfMonth: cronConfig.scheduleDayOfMonth ?? 1,
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
    this.emailForm.reset({
      folder: 'INBOX',
      fromAddress: '',
      toAddress: '',
      subjectPattern: '',
      markAsRead: false,
    });
    this.cronForm.reset({
      mode: 'interval',
      intervalValue: 30,
      intervalUnit: 'minutes',
      scheduleFrequency: 'daily',
      scheduleTime: '09:00',
      scheduleDayOfWeek: 1,
      scheduleDayOfMonth: 1,
    });
    this.selectedDbConnection.set(null);
    this.selectedEmailConnection.set(null);
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
      if (type === 'email') {
        return !!this.selectedEmailConnection();
      }
      if (type === 'cron') {
        const mode = this.cronForm.get('mode')?.value;
        if (mode === 'interval') {
          return !!this.cronForm.get('intervalValue')?.value && !!this.cronForm.get('intervalUnit')?.value;
        }
        if (mode === 'schedule') {
          return !!this.cronForm.get('scheduleFrequency')?.value && !!this.cronForm.get('scheduleTime')?.value;
        }
        return false;
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

    const mutation = this.sqlService.getTables(
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

    const mutation = this.sqlService.getColumns(
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

    if (type === 'email') {
      const email = this.selectedEmailConnection();
      const emailValues = this.emailForm.value;

      return {
        email: {
          metadataEmailId: email?.id,
          folder: emailValues.folder || 'INBOX',
          fromAddress: emailValues.fromAddress || undefined,
          toAddress: emailValues.toAddress || undefined,
          subjectPattern: emailValues.subjectPattern || undefined,
          markAsRead: emailValues.markAsRead || false,
        },
      };
    }

    if (type === 'cron') {
      const cronValues = this.cronForm.value;
      if (cronValues.mode === 'interval') {
        return {
          cron: {
            mode: 'interval',
            intervalValue: cronValues.intervalValue,
            intervalUnit: cronValues.intervalUnit,
          },
        };
      }
      return {
        cron: {
          mode: 'schedule',
          scheduleFrequency: cronValues.scheduleFrequency,
          scheduleTime: cronValues.scheduleTime,
          scheduleDayOfWeek: cronValues.scheduleFrequency === 'weekly' ? cronValues.scheduleDayOfWeek : undefined,
          scheduleDayOfMonth: cronValues.scheduleFrequency === 'monthly' ? cronValues.scheduleDayOfMonth : undefined,
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

  getEmailConnectionLabel(email: EmailMetadata): string {
    return `${email.name || email.username} (${email.imapHost}:${email.imapPort})`;
  }

  getEmailConnectionName(metadataId: number): string {
    const email = this.emailConnections().find(c => c.id === metadataId);
    return email ? `${email.name || email.username} (${email.imapHost}:${email.imapPort})` : `Connexion #${metadataId}`;
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

  getCronModeLabel(mode: string): string {
    return mode === 'interval' ? 'Intervalle' : 'Planifié';
  }

  getIntervalUnitLabel(unit: string): string {
    const labels: Record<string, string> = { minutes: 'minutes', hours: 'heures', days: 'jours' };
    return labels[unit] || unit;
  }

  getScheduleFrequencyLabel(freq: string): string {
    const labels: Record<string, string> = { daily: 'Tous les jours', weekly: 'Toutes les semaines', monthly: 'Tous les mois' };
    return labels[freq] || freq;
  }

  getDayOfWeekLabel(day: number): string {
    const labels = ['Dimanche', 'Lundi', 'Mardi', 'Mercredi', 'Jeudi', 'Vendredi', 'Samedi'];
    return labels[day] || '';
  }

  getCronSummary(): string {
    const cronValues = this.cronForm.value;
    if (cronValues.mode === 'interval') {
      return `Toutes les ${cronValues.intervalValue} ${this.getIntervalUnitLabel(cronValues.intervalUnit)}`;
    }
    let summary = `${this.getScheduleFrequencyLabel(cronValues.scheduleFrequency)} à ${cronValues.scheduleTime}`;
    if (cronValues.scheduleFrequency === 'weekly') {
      summary += ` (${this.getDayOfWeekLabel(cronValues.scheduleDayOfWeek)})`;
    }
    if (cronValues.scheduleFrequency === 'monthly') {
      summary += ` (le ${cronValues.scheduleDayOfMonth})`;
    }
    return summary;
  }

  // Notification contacts management
  loadNotificationContacts() {
    const trigger = this.selectedTrigger();
    if (!trigger) return;

    const contactsMap = new Map<number, NotificationContact[]>();
    for (const jobLink of trigger.jobs) {
      this.jobService.getById(jobLink.jobId, (data) => {
        contactsMap.set(jobLink.jobId, data.notificationContacts || []);
        this.notificationContacts.set(new Map(contactsMap));
      });
    }
  }

  getContactsForJob(jobId: number): NotificationContact[] {
    return this.notificationContacts().get(jobId) || [];
  }

  searchUsers(event: { query: string }) {
    this.userService.searchUsers(event.query, (users) => {
      this.filteredUsers.set(users);
    });
  }

  addNotificationContact(jobId: number, user: User) {
    const mutation = this.jobService.addNotificationContact(
      jobId,
      (updatedJob) => {
        const currentMap = this.notificationContacts();
        currentMap.set(jobId, updatedJob.notificationContacts || []);
        this.notificationContacts.set(new Map(currentMap));
        this.messageService.add({
          severity: 'success',
          summary: 'Succès',
          detail: `${user.prenom} ${user.nom} ajouté aux alertes`,
        });
      },
      () => {
        this.messageService.add({
          severity: 'error',
          summary: 'Erreur',
          detail: 'Impossible d\'ajouter le contact',
        });
      }
    );
    mutation.execute({ userId: user.id });
  }

  removeNotificationContact(jobId: number, contact: NotificationContact) {
    const mutation = this.jobService.removeNotificationContact(
      jobId,
      contact.id,
      (updatedJob) => {
        const currentMap = this.notificationContacts();
        currentMap.set(jobId, updatedJob.notificationContacts || []);
        this.notificationContacts.set(new Map(currentMap));
        this.messageService.add({
          severity: 'success',
          summary: 'Succès',
          detail: `${contact.prenom} ${contact.nom} retiré des alertes`,
        });
      },
      () => {
        this.messageService.add({
          severity: 'error',
          summary: 'Erreur',
          detail: 'Impossible de retirer le contact',
        });
      }
    );
    mutation.execute();
  }
}
