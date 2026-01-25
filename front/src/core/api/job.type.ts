// Enums
export type JobVisibility = 'public' | 'private';
export type OwningRole = 'owner' | 'editor' | 'viewer';
export type NodeType = 'start' | 'db_input' | 'db_output' | 'map';

// Port type
export interface Port {
  id: number;
  type: string;
  nodeId: number;
}

// Node type
export interface Node {
  id: number;
  type: NodeType;
  name: string;
  xpos: number;
  ypos: number;
  inputPort: Port[];
  outputPort: Port[];
  data: unknown;
  jobId: number;
}

// SharedUser - a user with access to a job
export interface SharedUser {
  id: number;
  email: string;
  prenom: string;
  nom: string;
  role: OwningRole;
}

// Job response without nodes (for listing)
export interface Job {
  id: number;
  name: string;
  description: string;
  filePath: string;
  creatorId: number;
  active: boolean;
  visibility: JobVisibility;
  outputPath: string;
  createdAt: string;
  updatedAt: string;
}

// Job response with nodes (for single job get)
export interface JobWithNodes extends Job {
  nodes: Node[];
  sharedWith?: SharedUser[];
}

// Request: Create a new job
export interface CreateJobRequest {
  name: string;
  description?: string;
  filePath?: string;
  outputPath?: string;
  active?: boolean;
  visibility?: JobVisibility;
  sharedWith?: number[];
}

// Request: Update an existing job
export interface UpdateJobRequest {
  name?: string;
  description?: string;
  filePath?: string;
  outputPath?: string;
  active?: boolean;
  visibility?: JobVisibility;
  sharedWith?: number[];
  nodes?: Node[];
}

// Request: Share/Unshare job
export interface ShareJobRequest {
  userIds: number[];
  role?: OwningRole;
}

// Response: Delete operation
export interface DeleteResponse {
  id: number;
  deleted: boolean;
}

// User type (mock for now)
export interface User {
  id: number;
  email: string;
  prenom: string;
  nom: string;
}

// Tree node for job explorer
export interface JobTreeNode {
  key: string;
  label: string;
  data?: Job;
  icon?: string;
  expandedIcon?: string;
  collapsedIcon?: string;
  children?: JobTreeNode[];
  leaf?: boolean;
  type: 'folder' | 'job';
}