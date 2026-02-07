// Trigger types matching backend models

export type TriggerType = 'database' | 'email' | 'webhook';
export type TriggerStatus = 'active' | 'paused' | 'error' | 'disabled';
export type WatermarkType = 'int' | 'timestamp' | 'uuid';
export type ExecutionStatus = 'running' | 'completed' | 'failed' | 'no_events';
export type ConditionOperator =
  | 'eq' | 'neq' | 'contains' | 'startsWith' | 'endsWith'
  | 'gt' | 'lt' | 'regex' | 'in' | 'notIn' | 'exists' | 'notExists';

// Database connection config
export interface DBConnectionConfig {
  type: 'postgres' | 'mysql' | 'sqlserver';
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
  sslMode?: string;
  dsn?: string;
}

// Database trigger configuration
export interface DatabaseTriggerConfig {
  metadataDatabaseId?: number;
  connection?: DBConnectionConfig;
  tableName: string;
  watermarkColumn: string;
  watermarkType: WatermarkType;
  lastWatermark?: string;
  selectColumns?: string[];
  whereClause?: string;
  batchSize?: number;
}

// Email trigger configuration
export interface EmailTriggerConfig {
  host: string;
  port: number;
  username: string;
  password: string;
  useTls: boolean;
  folder?: string;
  fromAddress?: string;
  toAddress?: string;
  subjectPattern?: string;
  hasAttachment?: boolean;
  ccAddresses?: string[];
  lastUid?: number;
  markAsRead?: boolean;
}

// Webhook trigger configuration
export interface WebhookTriggerConfig {
  secret?: string;
  requiredHeaders?: Record<string, string>;
}

// Combined trigger config
export interface TriggerConfig {
  database?: DatabaseTriggerConfig;
  email?: EmailTriggerConfig;
  webhook?: WebhookTriggerConfig;
}

// Rule condition
export interface RuleCondition {
  field: string;
  operator: ConditionOperator;
  value: unknown;
}

// Rule conditions (AND/OR logic)
export interface RuleConditions {
  all?: RuleCondition[];
  any?: RuleCondition[];
}

// Trigger rule
export interface TriggerRule {
  id: number;
  triggerId: number;
  name: string;
  conditions: RuleConditions;
  createdAt: string;
  updatedAt: string;
}

// Trigger-job link
export interface TriggerJobLink {
  id: number;
  triggerId: number;
  jobId: number;
  jobName: string;
  priority: number;
  active: boolean;
  passEventData: boolean;
}

// Trigger (list view)
export interface Trigger {
  id: number;
  name: string;
  description: string;
  type: TriggerType;
  status: TriggerStatus;
  creatorId: number;
  pollingInterval: number;
  lastPolledAt?: string;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
  jobCount: number;
}

// Trigger with full details
export interface TriggerWithDetails extends Trigger {
  config: TriggerConfig;
  rules: TriggerRule[];
  jobs: TriggerJobLink[];
}

// Trigger execution record
export interface TriggerExecution {
  id: number;
  triggerId: number;
  startedAt: string;
  finishedAt?: string;
  status: ExecutionStatus;
  eventCount: number;
  jobsTriggered: number;
  error?: string;
  eventSample?: string;
}

// Database introspection types
export interface DatabaseTable {
  schema: string;
  name: string;
}

export interface DatabaseColumn {
  name: string;
  dataType: string;
  isNullable: boolean;
  isPrimary: boolean;
}

export interface DatabaseIntrospection {
  tables?: DatabaseTable[];
  columns?: DatabaseColumn[];
}

export interface TestConnectionResult {
  success: boolean;
  message: string;
  version?: string;
}

// Request types
export interface CreateTriggerRequest {
  name: string;
  description?: string;
  type: TriggerType;
  pollingInterval?: number;
  config: TriggerConfig;
}

export interface UpdateTriggerRequest {
  name?: string;
  description?: string;
  pollingInterval?: number;
  config?: TriggerConfig;
}

export interface CreateTriggerRuleRequest {
  name?: string;
  conditions: RuleConditions;
}

export interface UpdateTriggerRuleRequest {
  name?: string;
  conditions?: RuleConditions;
}

export interface LinkJobRequest {
  jobId: number;
  priority?: number;
  passEventData?: boolean;
}

export interface TestConnectionRequest {
  connection: DBConnectionConfig;
}

export interface IntrospectDatabaseRequest {
  metadataDatabaseId?: number;
  connection?: DBConnectionConfig;
  tableName?: string; // For column introspection
}

// Response types
export interface DeleteResponse {
  id: number;
  deleted: boolean;
}

export interface UnlinkJobResponse {
  triggerId: number;
  jobId: number;
  unlinked: boolean;
}
