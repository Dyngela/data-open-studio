import {
  AfterViewInit,
  Component,
  computed,
  ElementRef,
  HostListener,
  inject,
  OnInit,
  signal,
  viewChild,
} from '@angular/core';
import {CommonModule} from '@angular/common';
import {ActivatedRoute, RouterOutlet} from '@angular/router';
import {CdkDragDrop, CdkDropList} from '@angular/cdk/drag-drop';
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
import {DbInputNodeConfig} from '../../../nodes/db-input/definition';
import {MapNodeConfig} from '../../../nodes/transform/definition';

@Component({
  selector: 'app-playground',
  standalone: true,
  imports: [CommonModule, CdkDropList, NodePanel, NodeInstanceComponent, Minimap, PlaygroundBottomBar, DbInputModal, StartModal, TransformModal, LogModal],
  templateUrl: './playground.html',
  styleUrl: './playground.css',
})
export class Playground implements OnInit, AfterViewInit {
  sidebarWidth = signal(250);
  isResizing = signal(false);

  leftTabs = signal([
    { label: 'Nodes', icon: '📁', active: true, position: 'left' as const },
    { label: 'Database', icon: '🗄️', active: false, position: 'left' as const },
    { label: 'Console', icon: '🖥️', active: false, position: 'bot' as const },
  ]);
  selectedTab = computed(() => this.leftTabs().find(t => t.active && t.position === 'left'));

  toggleSidebar(label: string, position: 'left' | 'bot') {
    if (label === 'reset') {
      this.leftTabs.update(tabs => tabs.map(t =>
        t.position === position ? { ...t, active: false } : t
      ));
      return;
    }
    this.leftTabs.update(tabs => tabs.map(t => {
      if (t.position !== position) return t;
      return { ...t, active: t.label === label ? !t.active : false };
    }));
  }

  startResizing(e: MouseEvent) {
    this.isResizing.set(true);
    e.preventDefault();
  }

  @HostListener('window:mousemove', ['$event'])
  onMouseMove(e: MouseEvent) {
    if (this.isResizing()) {
      const newWidth = e.clientX - 30; // 30px est la largeur de la barre d'icones
      if (newWidth > 100 && newWidth < 500) this.sidebarWidth.set(newWidth);
    }
  }

  @HostListener('window:mouseup')
  onMouseUp() {
    this.isResizing.set(false);
  }

  // Calculer si un panneau latéral est ouvert
  isAnySidePanelOpen = computed(() => this.leftTabs().some(t => t.active && t.label !== 'Console'));
  IsBottomPanelOpen = computed(() => this.leftTabs().some(t => t.active && t.label === 'Console'));

  private route = inject(ActivatedRoute);
  private jobService = inject(JobService);
  protected nodeGraph = inject(NodeGraphService);
  private jobState = inject(JobStateService);

  protected bottomBar = viewChild<PlaygroundBottomBar>('bottomBar')
  protected playgroundArea = viewChild<ElementRef>('playgroundArea')

  readonly nodes = this.nodeGraph.nodes;
  readonly connections = this.nodeGraph.connections;

  currentJobId = signal<number | null>(null);
  currentJob = signal<JobWithNodes | null>(null);
  isLoadingJob = signal(false);

  viewportWidth = signal(0);
  viewportHeight = signal(0);

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

  private isDraggingNode = signal(false);
  private draggedNodeId = signal<number | null>(null);
  private dragNodeOffset = signal({ x: 0, y: 0 });

  protected activeModal = signal<{ nodeId: number; nodeTypeId: string } | null>(null);

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
        this.viewportWidth.set(rect.width);
        this.viewportHeight.set(rect.height);
      }
    });
  }

  //#region Node callbacks
  onDbInputSave(nodeId: number, config: DbInputNodeConfig) {
    this.jobState.setNodeConfig(nodeId, config);
    this.closeModal();
  }

  onTransformSave(nodeId: number, config: MapNodeConfig) {
    this.jobState.setNodeConfig(nodeId, config);
    this.closeModal();
  }
  //#endregion

  //#region Mouse event handlers
  onNodeMouseDown(event: MouseEvent, nodeId: number) {
    if (this.isConnecting() || this.isPanning() || event.button !== 0) {
      return;
    }

    event.stopPropagation();
    this.isDraggingNode.set(true);
    this.draggedNodeId.set(nodeId);

    const node = this.nodeGraph.getNodeById(nodeId);
    if (node) {
      const playgroundRect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
      const offset = this.panOffset();

      const mouseCanvasX = event.clientX - playgroundRect.left - offset.x;
      const mouseCanvasY = event.clientY - playgroundRect.top - offset.y;

      this.dragNodeOffset.set({
        x: mouseCanvasX - node.position.x,
        y: mouseCanvasY - node.position.y,
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
        const dragOffset = this.dragNodeOffset();

        const mouseCanvasX = event.clientX - playgroundRect.left - offset.x;
        const mouseCanvasY = event.clientY - playgroundRect.top - offset.y;

        const newX = mouseCanvasX - dragOffset.x;
        const newY = mouseCanvasY - dragOffset.y;

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

      this.tempConnection.set({
        x1: sourcePortPos.x,
        y1: sourcePortPos.y,
        x2: event.clientX - playgroundRect.left,
        y2: event.clientY - playgroundRect.top,
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

    if (this.isConnecting()) {
      this.isConnecting.set(false);
      this.sourcePort.set(null);
      this.tempConnection.set(null);
    }
  }

  getConnectionPath(connection: Connection): string {
    return this.nodeGraph.getConnectionPath(
      connection,
      (nodeId, portIndex, portType, connectionType) =>
        this.getPortPosition(nodeId, portIndex, portType, connectionType),
    );
  }
  //#endregion

  //#region DOM access
  onDrop(event: CdkDragDrop<any>) {
    if (event.previousContainer.id === 'node-panel-list') {
      const nodeType = event.item.data as NodeType;
      const playgroundRect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
      const offset = this.panOffset();

      const dropX = event.dropPoint.x - playgroundRect.left - offset.x;
      const dropY = event.dropPoint.y - playgroundRect.top - offset.y;

      const position = this.nodeGraph.findNonOverlappingPosition(dropX, dropY);
      this.nodeGraph.createNode(nodeType, position);
    }
  }

  openNodeModal(nodeId: number) {
    const node = this.nodeGraph.getNodeById(nodeId);
    if (!node) return;

      this.activeModal.set({ nodeId: node.id, nodeTypeId: node.type.id });
  }

  closeModal() {
    this.activeModal.set(null);
  }

  private getNodeSize(nodeId: number): { width: number; height: number } {
    const nodeElement = this.playgroundArea()?.nativeElement.querySelector(
      `[data-node-id="${nodeId}"]`,
    ) as HTMLElement | null;
    if (nodeElement) {
      const rect = nodeElement.getBoundingClientRect();
      return { width: rect.width, height: rect.height };
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

    // Try DOM-based position first
    const portElement = this.getPortElement(nodeId, portIndex, portType, connectionType);
    if (portElement && this.playgroundArea) {
      const playgroundRect = this.playgroundArea()?.nativeElement.getBoundingClientRect();
      const portRect = portElement.getBoundingClientRect();

      return {
        x: portRect.left - playgroundRect.left + portRect.width / 2,
        y: portRect.top - playgroundRect.top + portRect.height / 2,
      };
    }

    // Fallback to calculated position
    return this.nodeGraph.calculatePortPosition(node, portIndex, portType, connectionType, this.panOffset());
  }

  private getPortElement(
    nodeId: number,
    portIndex: number,
    portType: Direction,
    connectionType: PortType = PortType.DATA,
  ): HTMLElement | null {
    const nodeElement = this.playgroundArea()?.nativeElement.querySelector(
      `[data-node-id="${nodeId}"]`,
    );
    if (!nodeElement) return null;

    const portSelector =
      portType === 'input'
        ? `.input-port.${connectionType}-port[data-port-index="${portIndex}"]`
        : `.output-port.${connectionType}-port[data-port-index="${portIndex}"]`;

    return nodeElement.querySelector(portSelector);
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

  onJobExecute() {
    const localConsole = this.bottomBar()?.getConsole();
    if (!localConsole) return;
    const id = this.currentJobId()
    if (!id) {
      return
    }
    const mutation = this.jobService.execute(id, () => {
        console.log("ok")
    },
      (error) => {
        console.log(error)
      }
      );
    mutation.execute(null);
  }

  onJobStop() {
    this.bottomBar()?.getConsole()?.addLog('warn', 'Exécution interrompue par l\'utilisateur');
  }
  //#endregion
}
