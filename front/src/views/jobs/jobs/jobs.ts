import { Component, inject, signal, computed, OnInit, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { TreeModule } from 'primeng/tree';
import { TreeNode, MenuItem } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { AutoCompleteModule, AutoCompleteCompleteEvent, AutoCompleteSelectEvent, AutoComplete } from 'primeng/autocomplete';
import { RadioButtonModule } from 'primeng/radiobutton';
import { TooltipModule } from 'primeng/tooltip';
import { ContextMenuModule } from 'primeng/contextmenu';
import { JobService } from '../../../core/api/job.service';
import {Job, JobVisibility, CreateJobRequest, User} from '../../../core/api/job.type';

@Component({
  selector: 'app-jobs',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    TreeModule,
    ButtonModule,
    DialogModule,
    InputTextModule,
    AutoCompleteModule,
    RadioButtonModule,
    TooltipModule,
    ContextMenuModule,
  ],
  templateUrl: './jobs.html',
  styleUrl: './jobs.css',
})
export class Jobs implements OnInit {
  private jobService = inject(JobService);
  private router = inject(Router);
  private fb = inject(FormBuilder);

  @ViewChild('userAutoComplete') userAutoComplete!: AutoComplete;

  // Mock users (à remplacer par API plus tard)
  mockUsers: User[] = [
    { id: 1, email: 'alice@example.com', prenom: 'Alice', nom: 'Martin' },
    { id: 2, email: 'bob@example.com', prenom: 'Bob', nom: 'Dupont' },
    { id: 3, email: 'claire@example.com', prenom: 'Claire', nom: 'Bernard' },
    { id: 4, email: 'david@example.com', prenom: 'David', nom: 'Petit' },
    { id: 5, email: 'emma@example.com', prenom: 'Emma', nom: 'Robert' },
  ];

  // Data
  jobsResult = this.jobService.getAll();
  jobs = computed(() => this.jobsResult.data() ?? []);
  isLoading = this.jobsResult.isLoading;

  // Tree data
  treeNodes = signal<TreeNode[]>([]);
  selectedNode = signal<TreeNode | null>(null);

  // Search
  searchQuery: string = '';
  filteredJobs = signal<Job[]>([]);

  // Create job modal
  showCreateModal = signal(false);
  selectedFolder = signal<string>('/');
  isSubmitting = signal(false);

  // User autocomplete
  filteredUsers = signal<User[]>([]);
  selectedUsers = signal<User[]>([]);

  // Context menu
  contextMenuItems: MenuItem[] = [];
  selectedContextNode = signal<TreeNode | null>(null);

  // Form
  createForm: FormGroup = this.fb.group({
    name: ['', [Validators.required, Validators.maxLength(100)]],
    description: ['', [Validators.maxLength(500)]],
    visibility: ['private' as JobVisibility, Validators.required],
  });

  ngOnInit() {
    // Build tree when jobs load
    this.buildTree();

    // Setup context menu
    this.contextMenuItems = [
      {
        label: 'Nouveau job',
        icon: 'pi pi-plus',
        command: () => this.openCreateModalInFolder(this.selectedContextNode()?.data?.path || '/'),
      },
      {
        label: 'Nouveau dossier',
        icon: 'pi pi-folder-plus',
        command: () => this.createFolder(),
      },
      { separator: true },
      {
        label: 'Renommer',
        icon: 'pi pi-pencil',
        command: () => this.renameItem(),
      },
      {
        label: 'Supprimer',
        icon: 'pi pi-trash',
        command: () => this.deleteItem(),
      },
    ];
  }

  buildTree() {
    const jobsList = this.jobs();
    const tree: TreeNode[] = [];
    const folderMap = new Map<string, TreeNode>();

    // Create root folder
    const rootFolder: TreeNode = {
      key: '/',
      label: 'Jobs',
      data: { path: '/', type: 'folder' },
      icon: 'pi pi-folder',
      expandedIcon: 'pi pi-folder-open',
      collapsedIcon: 'pi pi-folder',
      children: [],
      expanded: true,
    };
    folderMap.set('/', rootFolder);
    tree.push(rootFolder);

    // Process each job
    jobsList.forEach(job => {
      const filePath = job.filePath || '/';
      const pathParts = filePath.split('/').filter(p => p);

      // Create folder hierarchy
      let currentPath = '';
      let parentNode = rootFolder;

      pathParts.forEach(part => {
        currentPath += '/' + part;
        if (!folderMap.has(currentPath)) {
          const folderNode: TreeNode = {
            key: currentPath,
            label: part,
            data: { path: currentPath, type: 'folder' },
            icon: 'pi pi-folder',
            expandedIcon: 'pi pi-folder-open',
            collapsedIcon: 'pi pi-folder',
            children: [],
            expanded: true,
          };
          parentNode.children = parentNode.children || [];
          parentNode.children.push(folderNode);
          folderMap.set(currentPath, folderNode);
        }
        parentNode = folderMap.get(currentPath)!;
      });

      // Add job node
      const jobNode: TreeNode = {
        key: `job-${job.id}`,
        label: job.name,
        data: { ...job, type: 'job' },
        icon: job.visibility === 'public' ? 'pi pi-globe' : 'pi pi-lock',
        leaf: true,
      };
      parentNode.children = parentNode.children || [];
      parentNode.children.push(jobNode);
    });

    this.treeNodes.set(tree);
  }

  onNodeSelect(event: { node: TreeNode }) {
    this.selectedNode.set(event.node);
    if (event.node.data?.type === 'job') {
      this.openJob(event.node.data);
    }
  }

  onNodeContextMenu(event: { node: TreeNode }) {
    this.selectedContextNode.set(event.node);
  }

  openJob(job: Job) {
    this.router.navigate(['/playground', job.id]);
  }

  // Search
  searchJobs(event: AutoCompleteCompleteEvent) {
    const query = event.query.toLowerCase();
    const filtered = this.jobs().filter(job =>
      job.name.toLowerCase().includes(query) ||
      job.description?.toLowerCase().includes(query)
    );
    this.filteredJobs.set(filtered);
  }

  onJobSelect(event: AutoCompleteSelectEvent) {
    const job = event.value as Job;
    this.openJob(job);
  }

  // User autocomplete
  searchUsers(event: AutoCompleteCompleteEvent) {
    const query = event.query.toLowerCase();
    const selected = this.selectedUsers();
    const filtered = this.mockUsers.filter(user =>
      !selected.find(s => s.id === user.id) &&
      (user.email.toLowerCase().includes(query) ||
       user.prenom.toLowerCase().includes(query) ||
       user.nom.toLowerCase().includes(query) ||
       `${user.prenom} ${user.nom}`.toLowerCase().includes(query))
    );
    this.filteredUsers.set(filtered);
  }

  onUserSelect(event: AutoCompleteSelectEvent) {
    const user = event.value as User;
    this.selectedUsers.update(users => [...users, user]);
    // Clear the autocomplete input
    if (this.userAutoComplete) {
      this.userAutoComplete.clear();
    }
  }

  removeUser(user: User) {
    this.selectedUsers.update(users => users.filter(u => u.id !== user.id));
  }

  // Create job modal
  openCreateModal() {
    this.openCreateModalInFolder(this.selectedFolder());
  }

  openCreateModalInFolder(folderPath: string) {
    this.selectedFolder.set(folderPath);
    this.createForm.reset({
      name: '',
      description: '',
      visibility: 'private',
    });
    this.selectedUsers.set([]);
    this.showCreateModal.set(true);
  }

  closeCreateModal() {
    this.showCreateModal.set(false);
    this.createForm.reset();
    this.selectedUsers.set([]);
    if (this.userAutoComplete) {
      this.userAutoComplete.clear();
    }
  }

  onCreateJob() {
    if (this.createForm.invalid || this.isSubmitting()) {
      this.createForm.markAllAsTouched();
      return;
    }

    this.isSubmitting.set(true);
    const formValue = this.createForm.value;

    const request: CreateJobRequest = {
      name: formValue.name,
      description: formValue.description || undefined,
      filePath: this.selectedFolder(),
      visibility: formValue.visibility,
      sharedWith: this.selectedUsers().map(u => u.id),
      active: true,
    };

    const mutation = this.jobService.create(
      (job) => {
        this.isSubmitting.set(false);
        this.closeCreateModal();
        this.jobsResult.refresh();
        // Redirect to playground with job id
        this.router.navigate(['/playground', job.id]);
      },
      () => {
        this.isSubmitting.set(false);
      }
    );
    mutation.execute(request);
  }

  // Context menu actions
  createFolder() {
    const folderName = prompt('Nom du nouveau dossier:');
    if (folderName) {
      const parentPath = this.selectedContextNode()?.data?.path || '/';
      const newPath = parentPath === '/' ? `/${folderName}` : `${parentPath}/${folderName}`;

      // Add folder to tree (in real app, this would be stored)
      const parentNode = this.findNodeByPath(parentPath);
      if (parentNode) {
        const newFolder: TreeNode = {
          key: newPath,
          label: folderName,
          data: { path: newPath, type: 'folder' },
          icon: 'pi pi-folder',
          expandedIcon: 'pi pi-folder-open',
          collapsedIcon: 'pi pi-folder',
          children: [],
          expanded: true,
        };
        parentNode.children = parentNode.children || [];
        parentNode.children.push(newFolder);
        this.treeNodes.set([...this.treeNodes()]);
      }
    }
  }

  renameItem() {
    const node = this.selectedContextNode();
    if (!node) return;

    const newName = prompt('Nouveau nom:', node.label);
    if (newName && newName !== node.label) {
      node.label = newName;
      if (node.data?.type === 'job') {
        // Update job via API
        const job = node.data as Job;
        const mutation = this.jobService.update(job.id, () => {
          this.jobsResult.refresh();
        });
        mutation.execute({ name: newName });
      }
      this.treeNodes.set([...this.treeNodes()]);
    }
  }

  deleteItem() {
    const node = this.selectedContextNode();
    if (!node) return;

    if (node.data?.type === 'job') {
      const job = node.data as Job;
      if (confirm(`Supprimer le job "${job.name}" ?`)) {
        const mutation = this.jobService.deleteJob(job.id, () => {
          this.jobsResult.refresh();
          this.buildTree();
        });
        mutation.execute();
      }
    } else {
      if (confirm(`Supprimer le dossier "${node.label}" et son contenu ?`)) {
        // Remove folder from tree (would need to delete jobs in folder via API)
        this.removeNodeFromTree(node);
      }
    }
  }

  private findNodeByPath(path: string): TreeNode | null {
    const searchNode = (nodes: TreeNode[]): TreeNode | null => {
      for (const node of nodes) {
        if (node.data?.path === path) return node;
        if (node.children) {
          const found = searchNode(node.children);
          if (found) return found;
        }
      }
      return null;
    };
    return searchNode(this.treeNodes());
  }

  private removeNodeFromTree(nodeToRemove: TreeNode) {
    const removeFromChildren = (nodes: TreeNode[]): boolean => {
      const index = nodes.findIndex(n => n.key === nodeToRemove.key);
      if (index !== -1) {
        nodes.splice(index, 1);
        return true;
      }
      for (const node of nodes) {
        if (node.children && removeFromChildren(node.children)) {
          return true;
        }
      }
      return false;
    };
    removeFromChildren(this.treeNodes());
    this.treeNodes.set([...this.treeNodes()]);
  }

  // Helpers
  isFieldInvalid(fieldName: string): boolean {
    const field = this.createForm.get(fieldName);
    return !!(field && field.invalid && field.touched);
  }

  getFieldError(fieldName: string): string {
    const field = this.createForm.get(fieldName);
    if (!field?.errors) return '';
    if (field.errors['required']) return 'Ce champ est requis';
    if (field.errors['maxlength']) return 'Valeur trop longue';
    return 'Valeur invalide';
  }

  getUserDisplayName(user: User): string {
    return `${user.prenom} ${user.nom}`;
  }

  // Refresh tree when jobs change
  ngDoCheck() {
    // Simple check - in production use proper change detection
    if (this.jobs().length !== this.treeNodes()[0]?.children?.length) {
      this.buildTree();
    }
  }
}
