import {
  AfterViewChecked,
  AfterViewInit,
  Component,
  computed,
  ElementRef,
  HostListener,
  inject,
  OnDestroy,
  OnInit,
  signal,
  viewChild,
} from '@angular/core';
import {CommonModule} from '@angular/common';
import {ActivatedRoute, RouterOutlet} from '@angular/router';
import {CdkDragDrop, CdkDropList} from '@angular/cdk/drag-drop';
import {FormsModule} from '@angular/forms';
import {ContextMenu} from 'primeng/contextmenu';
import {MenuItem} from 'primeng/api';
import {NodePanel} from '../node-panel/node-panel';
import {NodeInstanceComponent} from '../node-instance/node-instance';
import {Minimap} from '../minimap/minimap';
import {PlaygroundBottomBar} from '../playground-bottom-bar/playground-bottom-bar';
import {Connection, Direction, NodeType, PortType, TempConnection} from '../../../core/nodes-services/node.type';
import {DbInputModal} from '../../../nodes/db-input/db-input.modal';
import {StartModal} from '../../../nodes/start/start.modal';
import {TransformModal} from '../../../nodes/transform/transform.modal';
import {LogModal} from '../../../nodes/log/log.modal';
import {JobService} from '../../../core/api/job.service';
import {JobWithNodes, UpdateJobRequest} from '../../../core/api/job.type';
import {NodeGraphService} from '../../../core/nodes-services/node-graph.service';
import {JobStateService} from '../../../core/nodes-services/job-state.service';
import {LayoutService} from '../../../core/services/layout-service';
import {JobRealtimeService} from '../../../core/services/base-ws.service';
import {NodeRegistryService} from '../../../nodes/node-registry.service';

@Component({
  selector: 'app-playground',
  standalone: true,
  imports: [CommonModule, CdkDropList, FormsModule, ContextMenu, NodePanel, NodeInstanceComponent, Minimap, PlaygroundBottomBar, DbInputModal, StartModal, TransformModal, LogModal],
  templateUrl: './playground.html',
  styleUrl: './playground.css',
})
export class Playground implements OnInit, AfterViewInit, AfterViewChecked, OnDestroy {

  @HostListener('window:mousemove', ['$event'])
  onMouseMove(e: MouseEvent) {
    if (this.layoutService.isResizing()) {
      const newWidth = e.clientX - 30; // 30px est la largeur de la barre d'icones
      if (newWidth > 100 && newWidth < 500) this.layoutService.sidebarWidth.set(newWidth);
    }
  }

  @HostListener('window:mouseup')
  onMouseUp() {
    this.layoutService.isResizing.set(false);
  }

  @HostListener('window:keydown', ['$event'])
  onKeyDown(e: KeyboardEvent) {
    if (e.key === 'Delete' && !this.isRenaming()) {
      if (this.selectedConnection()) {
        this.nodeGraph.deleteConnection(this.selectedConnection()!);
        this.selectedConnection.set(null);
      } else if (this.selectedNodeId() !== null) {
        this.nodeGraph.deleteNode(this.selectedNodeId()!);
        this.selectedNodeId.set(null);
      }
    }
  }

  private route = inject(ActivatedRoute);
  private jobService = inject(JobService);
  protected nodeGraph = inject(NodeGraphService);
  private jobState = inject(JobStateService);
  protected layoutService = inject(LayoutService);
  private realtime = inject(JobRealtimeService);
  private nodeRegistry = inject(NodeRegistryService);

  /** Track how many nodes are still running so we know when the job finishes */
  private runningNodes = new Set<number>();
  private readonly unsubProgress: () => void;

  constructor() {
    // React to every progress message: update node visual status and track completion
    this.unsubProgress = this.realtime.onProgress((progress) => {
      // nodeId === 0 is a job-level message (handled by Console), skip node updates
      if (progress.nodeId === 0) return;

      const nodeStatus = progress.status === 'running' ? 'running'
        : progress.status === 'completed' ? 'success'
        : 'error';
      this.nodeGraph.updateNodeStatus(progress.nodeId, nodeStatus);

      if (progress.status === 'running') {
        this.runningNodes.add(progress.nodeId);
      } else {
        this.runningNodes.delete(progress.nodeId);
      }
    });
  }

  protected bottomBar = viewChild<PlaygroundBottomBar>('bottomBar')
  protected playgroundArea = viewChild<ElementRef>('playgroundArea')
  protected contextMenu = viewChild<ContextMenu>('contextMenu');

  readonly nodes = this.nodeGraph.nodes;
  readonly connections = this.nodeGraph.connections;

  // Context menu state
  protected contextMenuItems = signal<MenuItem[]>([]);
  private contextWorldPos = signal<{ x: number; y: number }>({ x: 0, y: 0 });

  // Selection state
  protected selectedConnection = signal<Connection | null>(null);
  protected selectedNodeId = signal<number | null>(null);

  // Rename dialog state
  protected isRenaming = signal(false);
  protected renameValue = signal('');
  private renameNodeId = signal<number | null>(null);
  protected renamePosition = signal<{ x: number; y: number }>({ x: 0, y: 0 });

  currentJobId = signal<number | null>(null);
  currentJob = signal<JobWithNodes>({} as JobWithNodes);
  isLoadingJob = signal(false);



  // Pre-computed connection paths (updated after each render via DOM reading)
  protected connectionPathStrings = signal<string[]>([]);

  // Canvas interaction state
  private isConnecting = signal(false);
  private sourcePort = signal<{
    nodeId: number;
    portIndex: number;
    portType: PortType;
  } | null>(null);
  protected tempConnection = signal<TempConnection | null>(null);

  protected panOffset = signal({ x: 0, y: 0 });
  protected isPanning = signal(false);
  private panStart = signal({ x: 0, y: 0 });
  protected zoom = signal(1);

  protected canvasTransform = computed(
    () => `translate(${this.panOffset().x}px, ${this.panOffset().y}px) scale(${this.zoom()})`,
  );

  private isDraggingNode = signal(false);
  private draggedNodeId = signal<number | null>(null);
  private dragNodeOffset = signal({ x: 0, y: 0 });
  private pathUpdateScheduled = false;



  ngOnInit() {
    this.route.params.subscribe(params => {
      const jobId = params['id'];
      if (jobId) {
        this.loadJob(parseInt(jobId, 10));
      }
    });
  }

  ngAfterViewInit() {
    window.addEventListener('resize', () => {
      if (this.playgroundArea) {
        const rect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
        this.layoutService.viewportWidth.set(rect.width);
        this.layoutService.viewportHeight.set(rect.height);
      }
    });
  }


  ngAfterViewChecked() {
    if (this.pathUpdateScheduled) return;
    this.pathUpdateScheduled = true;
      const conns = this.connections();
      const newPaths = conns.map(conn =>
        this.nodeGraph.getConnectionPath(conn, (nodeId, portIndex, portType, connectionType) =>
          this.getPortPosition(nodeId, portIndex, portType, connectionType),
        ),
      );
      const current = this.connectionPathStrings();
      if (newPaths.length !== current.length || newPaths.some((p, i) => p !== current[i])) {
        this.connectionPathStrings.set(newPaths);
      }
    this.pathUpdateScheduled = false;
  }

  //#region Mouse event handlers
  onNodeMouseDown(event: MouseEvent, nodeId: number) {
    if (this.isConnecting() || this.isPanning() || event.button !== 0) {
      return;
    }

    event.stopPropagation();
    this.selectedNodeId.set(nodeId);
    this.selectedConnection.set(null);
    this.isDraggingNode.set(true);
    this.draggedNodeId.set(nodeId);

    const node = this.nodeGraph.getNodeById(nodeId);
    if (node) {
      const playgroundRect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
      const offset = this.panOffset();
      const z = this.zoom();

      const mouseWorldX = (event.clientX - playgroundRect.left - offset.x) / z;
      const mouseWorldY = (event.clientY - playgroundRect.top - offset.y) / z;

      this.dragNodeOffset.set({
        x: mouseWorldX - node.position.x,
        y: mouseWorldY - node.position.y,
      });
    }
  }

  onOutputPortClick(event: { nodeId: number; portIndex: number; portType: PortType }) {
    if (!this.isConnecting()) {
      this.isConnecting.set(true);
      this.sourcePort.set({
        nodeId: event.nodeId,
        portIndex: event.portIndex,
        portType: event.portType,
      });
    }
  }

  onInputPortClick(event: { nodeId: number; portIndex: number; portType: PortType }) {
    const source = this.sourcePort();
    if (this.isConnecting() && source) {
      this.nodeGraph.createConnection(source, {
        nodeId: event.nodeId,
        portIndex: event.portIndex,
        portType: event.portType,
      });
      this.isConnecting.set(false);
      this.sourcePort.set(null);
      this.tempConnection.set(null);
    }
  }

  onPlaygroundMouseDown(event: MouseEvent) {
    if (event.button === 1) {
      event.preventDefault();
      this.isPanning.set(true);
      this.panStart.set({ x: event.clientX, y: event.clientY });
    }
  }

  onPlaygroundMouseMove(event: MouseEvent) {
    // Priority 1: Node drag
    if (this.isDraggingNode()) {
      const nodeId = this.draggedNodeId();
      if (nodeId) {
        const playgroundRect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
        const offset = this.panOffset();
        const z = this.zoom();
        const dragOffset = this.dragNodeOffset();

        const mouseWorldX = (event.clientX - playgroundRect.left - offset.x) / z;
        const mouseWorldY = (event.clientY - playgroundRect.top - offset.y) / z;

        const newX = mouseWorldX - dragOffset.x;
        const newY = mouseWorldY - dragOffset.y;

        const adjusted = this.nodeGraph.resolveCollision(
          nodeId, newX, newY, (id) => this.getNodeSize(id),
        );
        this.nodeGraph.updateNodePosition(nodeId, adjusted);
      }
      return;
    }

    // Priority 2: Panning
    if (this.isPanning()) {
      const start = this.panStart();
      const offset = this.panOffset();
      const deltaX = event.clientX - start.x;
      const deltaY = event.clientY - start.y;

      this.panOffset.set({
        x: offset.x + deltaX,
        y: offset.y + deltaY,
      });

      this.panStart.set({ x: event.clientX, y: event.clientY });
      return;
    }

    // Priority 3: Temporary connection
    const source = this.sourcePort();
    if (this.isConnecting() && source) {
      const playgroundRect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
      const sourcePortPos = this.getPortPosition(
        source.nodeId,
        source.portIndex,
        Direction.OUTPUT,
        source.portType,
      );

      const z = this.zoom();
      const offset = this.panOffset();
      this.tempConnection.set({
        x1: sourcePortPos.x,
        y1: sourcePortPos.y,
        x2: (event.clientX - playgroundRect.left - offset.x) / z,
        y2: (event.clientY - playgroundRect.top - offset.y) / z,
      });
    }
  }

  onPlaygroundMouseUp(event: MouseEvent) {
    if (event.button === 1) {
      this.isPanning.set(false);
    }

    if (event.button === 0 && this.isDraggingNode()) {
      this.isDraggingNode.set(false);
      this.draggedNodeId.set(null);
    }
  }

  onPlaygroundClick(event: MouseEvent) {
    if (this.isPanning() || this.isDraggingNode()) {
      return;
    }

    this.selectedConnection.set(null);
    this.selectedNodeId.set(null);

    if (this.isConnecting()) {
      this.isConnecting.set(false);
      this.sourcePort.set(null);
      this.tempConnection.set(null);
    }
  }

  onWheel(event: WheelEvent) {
    event.preventDefault();
    if (this.layoutService.activeModal !== null) return;
    const oldZoom = this.zoom();
    const factor = event.deltaY < 0 ? 1.1 : 1 / 1.1;
    const newZoom = Math.min(Math.max(oldZoom * factor, 0.2), 5);

    const rect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
    if (!rect) return;

    const mouseX = event.clientX - rect.left;
    const mouseY = event.clientY - rect.top;
    const ratio = newZoom / oldZoom;
    const oldPan = this.panOffset();

    this.panOffset.set({
      x: mouseX - (mouseX - oldPan.x) * ratio,
      y: mouseY - (mouseY - oldPan.y) * ratio,
    });
    this.zoom.set(newZoom);
  }

  isConnectionSelected(connection: Connection): boolean {
    const sel = this.selectedConnection();
    if (!sel) return false;
    return sel.sourceNodeId === connection.sourceNodeId
      && sel.sourcePort === connection.sourcePort
      && sel.sourcePortType === connection.sourcePortType
      && sel.targetNodeId === connection.targetNodeId
      && sel.targetPort === connection.targetPort
      && sel.targetPortType === connection.targetPortType;
  }
  //#endregion

  //#region Context menu & selection
  onConnectionClick(event: MouseEvent, connection: Connection) {
    event.stopPropagation();
    this.selectedConnection.set(connection);
    this.selectedNodeId.set(null);
  }

  onConnectionContextMenu(event: MouseEvent, connection: Connection) {
    event.preventDefault();
    event.stopPropagation();

    this.selectedConnection.set(connection);

    this.contextMenuItems.set([
      {
        label: 'Delete Connection',
        icon: 'pi pi-trash',
        command: () => {
          this.nodeGraph.deleteConnection(connection);
          this.selectedConnection.set(null);
        },
      },
    ]);

    this.contextMenu()?.show(event);
  }

  onNodeContextMenu(event: MouseEvent, nodeId: number) {
    event.preventDefault();
    event.stopPropagation();

    this.contextMenuItems.set([
      {
        label: 'Open Settings',
        icon: 'pi pi-cog',
        command: () => this.layoutService.openNodeModal(nodeId),
      },
      {
        label: 'Rename',
        icon: 'pi pi-pencil',
        command: () => this.startRename(nodeId, event),
      },
      { separator: true },
      {
        label: 'Delete',
        icon: 'pi pi-trash',
        command: () => this.nodeGraph.deleteNode(nodeId),
      },
    ]);

    this.contextMenu()?.show(event);
  }

  onPlaygroundContextMenu(event: MouseEvent) {
    event.preventDefault();

    const playgroundRect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
    if (!playgroundRect) return;

    const offset = this.panOffset();
    const z = this.zoom();
    const worldX = (event.clientX - playgroundRect.left - offset.x) / z;
    const worldY = (event.clientY - playgroundRect.top - offset.y) / z;
    this.contextWorldPos.set({ x: worldX, y: worldY });

    const nodeTypes = this.nodeRegistry.getNodeTypes();
    const addNodeItems: MenuItem[] = nodeTypes.map(nt => ({
      label: nt.label,
      icon: nt.icon,
      command: () => {
        const position = this.nodeGraph.findNonOverlappingPosition(
          this.contextWorldPos().x,
          this.contextWorldPos().y,
        );
        this.nodeGraph.createNode(nt, position);
      },
    }));

    this.contextMenuItems.set([
      {
        label: 'Add Node',
        icon: 'pi pi-plus',
        items: addNodeItems,
      },
    ]);

    this.contextMenu()?.show(event);
  }

  private startRename(nodeId: number, event: MouseEvent) {
    const node = this.nodeGraph.getNodeById(nodeId);
    if (!node) return;

    this.renameNodeId.set(nodeId);
    this.renameValue.set(node.name || node.type.label);
    this.renamePosition.set({ x: event.clientX, y: event.clientY });
    this.isRenaming.set(true);
  }

  confirmRename() {
    const nodeId = this.renameNodeId();
    const value = this.renameValue().trim();
    if (nodeId !== null && value) {
      this.nodeGraph.renameNode(nodeId, value);
    }
    this.cancelRename();
  }

  cancelRename() {
    this.isRenaming.set(false);
    this.renameNodeId.set(null);
    this.renameValue.set('');
  }
  //#endregion

  //#region DOM access
  onDrop(event: CdkDragDrop<any>) {
    if (event.previousContainer.id === 'node-panel-list') {
      const nodeType = event.item.data as NodeType;
      const playgroundRect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
      const offset = this.panOffset();
      const z = this.zoom();

      const dropX = (event.dropPoint.x - playgroundRect.left - offset.x) / z;
      const dropY = (event.dropPoint.y - playgroundRect.top - offset.y) / z;

      const position = this.nodeGraph.findNonOverlappingPosition(dropX, dropY);
      this.nodeGraph.createNode(nodeType, position);
    }
  }



  private getNodeSize(nodeId: number): { width: number; height: number } {
    const nodeElement = this.playgroundArea()?.nativeElement.querySelector(
      `.node-instance[data-node-id="${nodeId}"]`,
    ) as HTMLElement | null;
    if (nodeElement) {
      const rect = nodeElement.getBoundingClientRect();
      const z = this.zoom();
      return { width: rect.width / z, height: rect.height / z };
    }
    return {
      width: this.nodeGraph.NODE_DIMENSIONS.width,
      height: this.nodeGraph.NODE_DIMENSIONS.estimatedHeight,
    };
  }

  private getPortPosition(
    nodeId: number,
    portIndex: number,
    portType: Direction,
    connectionType: PortType = PortType.DATA,
  ): { x: number; y: number } {
    const node = this.nodeGraph.getNodeById(nodeId);
    if (!node) return { x: 0, y: 0 };

    const playground = this.playgroundArea()?.nativeElement;
    if (!playground) {
      return this.nodeGraph.calculatePortPosition(node, portIndex, portType, connectionType);
    }

    // Build a unique selector that directly targets the port element
    const dirClass = portType === 'input' ? 'input-port' : 'output-port';
    const typeClass = `${connectionType}-port`;
    const selector =
      `.node-instance[data-node-id="${nodeId}"] .${dirClass}.${typeClass}[data-port-index="${portIndex}"]`;

    const portElement = playground.querySelector(selector) as HTMLElement | null;
    if (!portElement) {
      return this.nodeGraph.calculatePortPosition(node, portIndex, portType, connectionType);
    }

    const playgroundRect = playground.getBoundingClientRect();
    const portRect = portElement.getBoundingClientRect();
    const z = this.zoom();
    const offset = this.panOffset();

    return {
      x: (portRect.left + portRect.width / 2 - playgroundRect.left - offset.x) / z,
      y: (portRect.top + portRect.height / 2 - playgroundRect.top - offset.y) / z,
    };
  }

  //#endregion

  //#region Job
  loadJob(jobId: number) {
    this.isLoadingJob.set(true);
    this.currentJobId.set(jobId);

    const result = this.jobService.getById(jobId);

    const checkLoaded = setInterval(() => {
      if (!result.isLoading()) {
        clearInterval(checkLoaded);
        this.isLoadingJob.set(false);

        const job = result.data();
        if (job) {
          this.currentJob.set(job);
          this.nodeGraph.loadFromJob(job);
          this.jobState.loadFromNodes(this.nodeGraph.nodes());
        }
      }
    }, 100);
  }

  onJobSave() {
    const localConsole = this.bottomBar()?.getConsole();
    const jobId = this.currentJobId();

    if (!jobId) {
      localConsole?.addLog('warn', 'Aucun job chargé. Impossible de sauvegarder.');
      localConsole?.isSaving.set(false);
      return;
    }

    localConsole?.isSaving.set(true);
    localConsole?.addLog('info', 'Sauvegarde du job en cours...');

    const apiNodes = this.nodeGraph.toApiNodes(jobId);

    const request: UpdateJobRequest = {
      nodes: apiNodes,
      connexions: this.nodeGraph.connections(),
    };

    const mutation = this.jobService.update(
      jobId,
      (updatedJob) => {
        this.currentJob.set(updatedJob);
        localConsole?.addLog('success', 'Job sauvegardé avec succès.');
        localConsole?.isSaving.set(false);
      },
      () => {
        localConsole?.addLog('error', 'Erreur lors de la sauvegarde du job.');
        localConsole?.isSaving.set(false);
      }
    );

    mutation.execute(request);
  }

  async onJobExecute() {
    const localConsole = this.bottomBar()?.getConsole();
    if (!localConsole) return;
    const id = this.currentJobId();
    if (!id) return;

    // Reset node statuses
    this.runningNodes.clear();
    for (const node of this.nodeGraph.nodes()) {
      this.nodeGraph.updateNodeStatus(node.id, 'idle');
    }

    // Wait for WebSocket to connect and subscribe before triggering execution
    localConsole.addLog('info', 'Connexion au flux temps réel...');
    // try {
    //   await this.realtime.subscribeToJob(id);
    // } catch {
    //   localConsole.addLog('warn', 'WebSocket non disponible, lancement sans suivi temps réel');
    // }

    const mutation = this.jobService.execute(id,
      () => {
        localConsole.addLog('info', 'Job lancé, en attente des résultats...');
        localConsole.setRunning(true);
      },
      (error) => {
        localConsole.markError(error?.message ?? 'Erreur lors du lancement du job');
        localConsole.setRunning(false);
      },
    );
    mutation.execute(null);
  }

  onJobStop() {
    const localConsole = this.bottomBar()?.getConsole();
    localConsole?.addLog('warn', 'Exécution interrompue par l\'utilisateur');
    this.realtime.disconnect();
    localConsole?.setRunning(false);
  }

  ngOnDestroy(): void {
    this.unsubProgress();
    this.realtime.disconnect();
  }
  //#endregion
}
