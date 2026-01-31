import {
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

@Component({
  selector: 'app-playground',
  standalone: true,
  imports: [CommonModule, CdkDropList, NodePanel, NodeInstanceComponent, Minimap, PlaygroundBottomBar, DbInputModal, StartModal, TransformModal, LogModal],
  templateUrl: './playground.html',
  styleUrl: './playground.css',
})
export class Playground implements OnInit, AfterViewInit, OnDestroy {

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

  private route = inject(ActivatedRoute);
  private jobService = inject(JobService);
  protected nodeGraph = inject(NodeGraphService);
  private jobState = inject(JobStateService);
  protected layoutService = inject(LayoutService);
  private realtime = inject(JobRealtimeService);

  /** Track how many nodes are still running so we know when the job finishes */
  private runningNodes = new Set<number>();
  private unsubProgress: () => void;

  constructor() {
    // React to every progress message: update node visual status and track completion
    this.unsubProgress = this.realtime.onProgress((progress) => {
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

  readonly nodes = this.nodeGraph.nodes;
  readonly connections = this.nodeGraph.connections;

  currentJobId = signal<number | null>(null);
  currentJob = signal<JobWithNodes | null>(null);
  isLoadingJob = signal(false);



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
    try {
      await this.realtime.subscribeToJob(id);
    } catch {
      localConsole.addLog('warn', 'WebSocket non disponible, lancement sans suivi temps réel');
    }

    // const mutation = this.jobService.execute(id,
    //   () => {
    //     localConsole.addLog('info', 'Job lancé, en attente des résultats...');
    //   },
    //   (error) => {
    //     localConsole.markError(error?.message ?? 'Erreur lors du lancement du job');
    //   },
    // );
    // mutation.execute(null);
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
