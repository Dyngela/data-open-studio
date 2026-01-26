import {
  Component,
  ViewChild,
  ElementRef,
  AfterViewInit,
  OnInit,
  signal,
  inject,
  HostListener,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute } from '@angular/router';
import { CdkDropList, CdkDragDrop } from '@angular/cdk/drag-drop';
import { NodePanel } from '../node-panel/node-panel';
import { NodeInstanceComponent } from '../node-instance/node-instance';
import { Minimap } from '../minimap/minimap';
import { PlaygroundBottomBar } from '../playground-bottom-bar/playground-bottom-bar';
import { NodeInstance, Connection, NodeType } from '../../../core/services/node.type';
import { DbInputModal } from '../../nodes/db-input/db-input.modal';
import { StartModal } from '../../nodes/start/start.modal';
import { TransformModal } from '../../nodes/transform/transform.modal';
import { JobService } from '../../../core/api/job.service';
import { JobWithNodes } from '../../../core/api/job.type';

@Component({
  selector: 'app-playground',
  standalone: true,
  imports: [CommonModule, CdkDropList, NodePanel, NodeInstanceComponent, Minimap, PlaygroundBottomBar, DbInputModal, StartModal, TransformModal],
  templateUrl: './playground.html',
  styleUrl: './playground.css',
})
export class Playground implements OnInit, AfterViewInit {
  private route = inject(ActivatedRoute);
  private jobService = inject(JobService);

  @ViewChild('playgroundArea', { static: false }) playgroundArea!: ElementRef;
  @ViewChild('fileInput', { static: false }) fileInput!: ElementRef<HTMLInputElement>;
  @ViewChild('bottomBar') bottomBar?: PlaygroundBottomBar;

  // Current job
  currentJobId = signal<number | null>(null);
  currentJob = signal<JobWithNodes | null>(null);
  isLoadingJob = signal(false);

  nodes = signal<NodeInstance[]>([]);
  connections = signal<Connection[]>([]); // Liste unifiée pour data et flow

  viewportWidth = signal(0);
  viewportHeight = signal(0);

  // Constantes de layout des nodes (basées sur node-instance.css)
  private readonly NODE_DIMENSIONS = {
    width: 180, // min-width du .node-instance
    headerPadding: 12, // padding: 0.75rem (0.75 * 16 = 12px)
    bodyPadding: 16, // padding: 1rem du .node-body
    portSize: 16, // width/height du .port
    portBorder: 2, // border: 2px du .port
    portGap: 12, // gap: 0.75rem entre les ports
    portOffset: 10, // left/right: -10px pour les ports
    estimatedHeight: 140, // hauteur estimée pour collision fallback
  };

  private readonly COLLISION_PADDING = 8; // marge pour empêcher le chevauchement

  private nodeIdCounter = 0;
  private connectionIdCounter = 0;

  private isConnecting = signal(false);
  private sourcePort = signal<{
    nodeId: string;
    portIndex: number;
    portType: 'data' | 'flow';
  } | null>(null);
  tempConnection = signal<{ x1: number; y1: number; x2: number; y2: number } | null>(null);

  panOffset = signal({ x: 0, y: 0 });
  protected isPanning = signal(false);
  private panStart = signal({ x: 0, y: 0 });

  private isDraggingNode = signal(false);
  private draggedNodeId = signal<string | null>(null);
  private dragNodeOffset = signal({ x: 0, y: 0 });

  protected activeModal = signal<{ nodeId: string; nodeTypeId: string } | null>(null);

  ngOnInit() {
    // Load job from route params
    this.route.params.subscribe(params => {
      const jobId = params['id'];
      if (jobId) {
        this.loadJob(parseInt(jobId, 10));
      }
    });
  }

  ngAfterViewInit() {
    this.updateViewportSize();
    window.addEventListener('resize', () => this.updateViewportSize());
  }

  loadJob(jobId: number) {
    this.isLoadingJob.set(true);
    this.currentJobId.set(jobId);

    const result = this.jobService.getById(jobId);

    // Watch for data changes
    const checkLoaded = setInterval(() => {
      if (!result.isLoading()) {
        clearInterval(checkLoaded);
        this.isLoadingJob.set(false);

        const job = result.data();
        if (job) {
          this.currentJob.set(job);
          this.loadJobNodes(job);
        }
      }
    }, 100);
  }

  loadJobNodes(job: JobWithNodes) {
    // Convert API nodes to NodeInstance format
    if (job.nodes && job.nodes.length > 0) {
      const nodeInstances: NodeInstance[] = job.nodes.map(apiNode => ({
        id: `node-${apiNode.id}`,
        type: this.getNodeTypeFromApiType(apiNode.type),
        position: { x: apiNode.xpos, y: apiNode.ypos },
        config: apiNode.data as Record<string, any> || {},
        status: 'idle' as const,
      }));
      this.nodes.set(nodeInstances);
      this.nodeIdCounter = Math.max(...job.nodes.map(n => n.id)) + 1;
    }
  }

  private getNodeTypeFromApiType(apiType: string): NodeType {
    // Map API node types to frontend NodeType
    const typeMap: Record<string, NodeType> = {
      'start': { id: 'start', label: 'Start', icon: 'pi-play', type: 'start', hasFlowInput: false, hasFlowOutput: true, hasDataInput: false, hasDataOutput: false },
      'db_input': { id: 'db-input', label: 'DB Input', icon: 'pi-database', type: 'input', hasFlowInput: true, hasFlowOutput: true, hasDataInput: false, hasDataOutput: true },
      'db_output': { id: 'db-output', label: 'DB Output', icon: 'pi-database', type: 'output', hasFlowInput: true, hasFlowOutput: true, hasDataInput: true, hasDataOutput: false },
      'map': { id: 'transform', label: 'Transform', icon: 'pi-cog', type: 'process', hasFlowInput: true, hasFlowOutput: true, hasDataInput: true, hasDataOutput: true },
    };
    return typeMap[apiType] || typeMap['start'];
  }

  private updateViewportSize() {
    if (this.playgroundArea) {
      const rect = this.playgroundArea.nativeElement.getBoundingClientRect();
      this.viewportWidth.set(rect.width);
      this.viewportHeight.set(rect.height);
    }
  }

  onDrop(event: CdkDragDrop<any>) {
    if (event.previousContainer.id === 'node-panel-list') {
      const nodeType = event.item.data as NodeType;
      const playgroundRect = this.playgroundArea.nativeElement.getBoundingClientRect();
      const offset = this.panOffset();

      let dropX = event.dropPoint.x - playgroundRect.left - offset.x;
      let dropY = event.dropPoint.y - playgroundRect.top - offset.y;

      const position = this.findNonOverlappingPosition(dropX, dropY);

      const newNode: NodeInstance = {
        id: `node-${this.nodeIdCounter++}`,
        type: nodeType,
        position: position,
        config: {},
        status: 'idle',
      };

      this.nodes.update((nodes) => [...nodes, newNode]);
    }
  }

  openNodeModal(nodeId: string) {
    const node = this.nodes().find((n) => n.id === nodeId);
    if (!node) return;

    if (node.type.id === 'db-input') {
      this.activeModal.set({ nodeId: node.id, nodeTypeId: node.type.id });
    }
    if (node.type.id === 'transform') {
      this.activeModal.set({ nodeId: node.id, nodeTypeId: node.type.id });
    }
    if (node.type.id === 'start') {
      this.activeModal.set({ nodeId: node.id, nodeTypeId: node.type.id });
    }
  }

  closeModal() {
    this.activeModal.set(null);
  }

  onDbInputSave(
    nodeId: string,
    config: {
      connectionString: string;
      table: string;
      query: string;
      database?: string;
      connectionId?: string;
      dbType?: string;
      host?: string;
      port?: string;
      username?: string;
      password?: string;
      sslMode?: string;
    },
  ) {
    this.nodes.update((nodes) =>
      nodes.map((node) =>
        node.id === nodeId ? { ...node, config: { ...node.config, ...config } } : node,
      ),
    );
    this.closeModal();
  }

  // Drag manuel des nodes
  onNodeMouseDown(event: MouseEvent, nodeId: string) {
    if (this.isConnecting() || this.isPanning() || event.button !== 0) {
      return;
    }

    event.stopPropagation();
    this.isDraggingNode.set(true);
    this.draggedNodeId.set(nodeId);

    const node = this.nodes().find((n) => n.id === nodeId);
    if (node) {
      const playgroundRect = this.playgroundArea.nativeElement.getBoundingClientRect();
      const offset = this.panOffset();

      // Position de la souris dans l'espace canvas
      const mouseCanvasX = event.clientX - playgroundRect.left - offset.x;
      const mouseCanvasY = event.clientY - playgroundRect.top - offset.y;

      // Offset entre la souris et le coin du node
      this.dragNodeOffset.set({
        x: mouseCanvasX - node.position.x,
        y: mouseCanvasY - node.position.y,
      });
    }
  }

  onOutputPortClick(event: { nodeId: string; portIndex: number; portType: 'data' | 'flow' }) {
    if (!this.isConnecting()) {
      this.isConnecting.set(true);
      this.sourcePort.set({
        nodeId: event.nodeId,
        portIndex: event.portIndex,
        portType: event.portType,
      });
    }
  }

  onInputPortClick(event: { nodeId: string; portIndex: number; portType: 'data' | 'flow' }) {
    const source = this.sourcePort();
    if (this.isConnecting() && source) {
      const newConnection: Connection = {
        id: `connection-${this.connectionIdCounter++}`,
        sourceNodeId: source.nodeId,
        sourcePort: source.portIndex,
        sourcePortType: source.portType,
        targetNodeId: event.nodeId,
        targetPort: event.portIndex,
        targetPortType: event.portType,
      };

      this.connections.update((connections) => [...connections, newConnection]);
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
    // Priorité 1: Drag de node
    if (this.isDraggingNode()) {
      const nodeId = this.draggedNodeId();
      if (nodeId) {
        const playgroundRect = this.playgroundArea.nativeElement.getBoundingClientRect();
        const offset = this.panOffset();
        const dragOffset = this.dragNodeOffset();

        // Position de la souris dans l'espace canvas
        const mouseCanvasX = event.clientX - playgroundRect.left - offset.x;
        const mouseCanvasY = event.clientY - playgroundRect.top - offset.y;

        // Nouvelle position du node = souris dans canvas - offset initial
        const newX = mouseCanvasX - dragOffset.x;
        const newY = mouseCanvasY - dragOffset.y;

        // Empêche les chevauchements : on glisse au plus proche sans bloquer
        const adjusted = this.resolveCollision(nodeId, newX, newY);
        this.nodes.update((nodes) =>
          nodes.map((node) =>
            node.id === nodeId ? { ...node, position: { x: adjusted.x, y: adjusted.y } } : node,
          ),
        );
      }
      return;
    }

    // Priorité 2: Panning
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

    // Priorité 3: Connection temporaire
    const source = this.sourcePort();
    if (this.isConnecting() && source) {
      const playgroundRect = this.playgroundArea.nativeElement.getBoundingClientRect();
      const sourcePortPos = this.getPortPosition(
        source.nodeId,
        source.portIndex,
        'output',
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

  protected getNodeById(nodeId: string): NodeInstance | undefined {
    return this.nodes().find((n) => n.id === nodeId);
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
    const portType = connection.sourcePortType;
    const sourcePos = this.getPortPosition(
      connection.sourceNodeId,
      connection.sourcePort,
      'output',
      portType,
    );
    const targetPos = this.getPortPosition(
      connection.targetNodeId,
      connection.targetPort,
      'input',
      connection.targetPortType,
    );

    const dx = targetPos.x - sourcePos.x;
    const controlPointOffset = Math.abs(dx) * 0.5;

    return `M ${sourcePos.x} ${sourcePos.y} C ${sourcePos.x + controlPointOffset} ${sourcePos.y}, ${
      targetPos.x - controlPointOffset
    } ${targetPos.y}, ${targetPos.x} ${targetPos.y}`;
  }

  getTempConnectionPath(): string {
    const temp = this.tempConnection();
    if (!temp) return '';

    const dx = temp.x2 - temp.x1;
    const controlPointOffset = Math.abs(dx) * 0.5;

    return `M ${temp.x1} ${temp.y1} C ${temp.x1 + controlPointOffset} ${temp.y1}, ${
      temp.x2 - controlPointOffset
    } ${temp.y2}, ${temp.x2} ${temp.y2}`;
  }

  getConnectionKind(connection: Connection): 'data' | 'flow' {
    return connection.sourcePortType || 'data';
  }

  private getNodeSize(nodeId: string): { width: number; height: number } {
    const nodeElement = this.playgroundArea?.nativeElement.querySelector(
      `[data-node-id="${nodeId}"]`,
    ) as HTMLElement | null;
    if (nodeElement) {
      const rect = nodeElement.getBoundingClientRect();
      return { width: rect.width, height: rect.height };
    }
    return { width: this.NODE_DIMENSIONS.width, height: this.NODE_DIMENSIONS.estimatedHeight };
  }

  private resolveCollision(
    nodeId: string,
    desiredX: number,
    desiredY: number,
  ): { x: number; y: number } {
    let x = desiredX;
    let y = desiredY;
    const padding = this.COLLISION_PADDING;
    const movingSize = this.getNodeSize(nodeId);

    // Iterative nudge to find nearest non-overlapping spot
    for (let i = 0; i < 10; i++) {
      const movingRect = {
        x: x - padding,
        y: y - padding,
        width: movingSize.width + padding * 2,
        height: movingSize.height + padding * 2,
      };

      let adjusted = false;

      for (const node of this.nodes()) {
        if (node.id === nodeId) continue;
        const size = this.getNodeSize(node.id);
        const otherRect = {
          x: node.position.x,
          y: node.position.y,
          width: size.width,
          height: size.height,
        };

        const overlapX =
          Math.min(movingRect.x + movingRect.width, otherRect.x + otherRect.width) -
          Math.max(movingRect.x, otherRect.x);
        const overlapY =
          Math.min(movingRect.y + movingRect.height, otherRect.y + otherRect.height) -
          Math.max(movingRect.y, otherRect.y);

        if (overlapX > 0 && overlapY > 0) {
          // Choose the smallest push axis
          if (overlapX < overlapY) {
            const movingCenterX = movingRect.x + movingRect.width / 2;
            const otherCenterX = otherRect.x + otherRect.width / 2;
            if (movingCenterX < otherCenterX) {
              x -= overlapX; // push left
            } else {
              x += overlapX; // push right
            }
          } else {
            const movingCenterY = movingRect.y + movingRect.height / 2;
            const otherCenterY = otherRect.y + otherRect.height / 2;
            if (movingCenterY < otherCenterY) {
              y -= overlapY; // push up
            } else {
              y += overlapY; // push down
            }
          }

          adjusted = true;
          break; // Re-evaluate after first adjustment
        }
      }

      if (!adjusted) {
        return { x, y };
      }
    }

    return { x, y };
  }

  /**
   * Récupère la position réelle d'un port en utilisant le DOM si possible,
   * sinon calcule la position théorique
   */
  private getPortPosition(
    nodeId: string,
    portIndex: number,
    portType: 'input' | 'output',
    connectionType: 'data' | 'flow' = 'data',
  ): { x: number; y: number } {
    const node = this.nodes().find((n) => n.id === nodeId);
    if (!node) return { x: 0, y: 0 };

    // Essayer d'obtenir la position réelle depuis le DOM
    const portElement = this.getPortElement(nodeId, portIndex, portType, connectionType);
    if (portElement && this.playgroundArea) {
      const playgroundRect = this.playgroundArea.nativeElement.getBoundingClientRect();
      const portRect = portElement.getBoundingClientRect();

      return {
        x: portRect.left - playgroundRect.left + portRect.width / 2,
        y: portRect.top - playgroundRect.top + portRect.height / 2,
      };
    }

    // Fallback sur le calcul théorique
    return this.calculatePortPosition(node, portIndex, portType, connectionType);
  }

  /**
   * Récupère l'élément DOM d'un port
   */
  private getPortElement(
    nodeId: string,
    portIndex: number,
    portType: 'input' | 'output',
    connectionType: 'data' | 'flow' = 'data',
  ): HTMLElement | null {
    const nodeElement = this.playgroundArea?.nativeElement.querySelector(
      `[data-node-id="${nodeId}"]`,
    );
    if (!nodeElement) return null;

    const portSelector =
      portType === 'input'
        ? `.input-port.${connectionType}-port[data-port-index="${portIndex}"]`
        : `.output-port.${connectionType}-port[data-port-index="${portIndex}"]`;

    return nodeElement.querySelector(portSelector);
  }

  /**
   * Calcule la position théorique d'un port basée sur les constantes CSS
   * Note: Les data ports utilisent position:absolute + top:50% + translateY(-50%)
   * donc ils sont automatiquement centrés verticalement dans le body
   * Les flow ports sont positionnés dans le header
   */
  private calculatePortPosition(
    node: NodeInstance,
    portIndex: number,
    portType: 'input' | 'output',
    connectionType: 'data' | 'flow' = 'data',
  ): { x: number; y: number } {
    const dim = this.NODE_DIMENSIONS;
    const offset = this.panOffset();

    // Hauteur estimée du header (padding + icône + texte)
    const estimatedHeaderHeight = dim.headerPadding * 2 + 16; // ~40px

    // Traitement différent pour flow (header) et data (body)
    if (connectionType === 'flow') {
      // Flow ports sont dans le header, centrés verticalement
      const portY = estimatedHeaderHeight / 2;

      // Position X du centre du port
      // input: left: -10px → centre à -10 + 8 = -2
      // output: right: -10px → centre à width + 10 - 8 = width + 2
      const portCenterOffset = dim.portSize / 2;
      const portX =
        portType === 'input'
          ? -dim.portOffset + portCenterOffset
          : dim.width + dim.portOffset - portCenterOffset;

      return {
        x: node.position.x + portX + offset.x,
        y: node.position.y + portY + offset.y,
      };
    } else {
      // Data ports - logique existante dans le body
      // Déterminer le nombre de ports selon le type
      const portCount =
        portType === 'input' ? (node.type.hasDataInput ? 1 : 0) : node.type.hasDataOutput ? 1 : 0;

      // Centre du body
      const bodyTop = estimatedHeaderHeight + dim.bodyPadding;

      // Les ports sont centrés verticalement avec top: 50% + translateY(-50%)
      // Calcul de la position Y du premier port
      const totalPortsHeight = portCount * dim.portSize + (portCount - 1) * dim.portGap;
      const portsStartY = bodyTop - totalPortsHeight / 2;
      const portY = portsStartY + portIndex * (dim.portSize + dim.portGap) + dim.portSize / 2;

      // Position X du centre du port
      const portCenterOffset = dim.portSize / 2;
      const portX =
        portType === 'input'
          ? -dim.portOffset + portCenterOffset
          : dim.width + dim.portOffset - portCenterOffset;

      return {
        x: node.position.x + portX + offset.x,
        y: node.position.y + portY + offset.y,
      };
    }
  }

  private findNonOverlappingPosition(x: number, y: number): { x: number; y: number } {
    const nodeWidth = this.NODE_DIMENSIONS.width;
    const nodeHeight = 100;
    const minDistance = 20;

    let finalX = x;
    let finalY = y;
    let attempts = 0;
    const maxAttempts = 50;

    while (attempts < maxAttempts) {
      let overlapping = false;

      for (const node of this.nodes()) {
        const dx = Math.abs(finalX - node.position.x);
        const dy = Math.abs(finalY - node.position.y);

        if (dx < nodeWidth + minDistance && dy < nodeHeight + minDistance) {
          overlapping = true;
          break;
        }
      }

      if (!overlapping) {
        break;
      }

      finalX += 30;
      finalY += 30;
      attempts++;
    }

    return { x: finalX, y: finalY };
  }

  // Sauvegarde et chargement de schéma
  saveSchema() {
    const schema = {
      nodes: this.nodes(),
      connections: this.connections(),
      panOffset: this.panOffset(),
      nodeIdCounter: this.nodeIdCounter,
      connectionIdCounter: this.connectionIdCounter,
      version: '3.0', // Nouvelle version avec connexions unifiées
    };

    const blob = new Blob([JSON.stringify(schema, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `schema-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`;
    link.click();
    URL.revokeObjectURL(url);
  }

  loadSchema() {
    this.fileInput.nativeElement.click();
  }

  onFileSelected(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    const reader = new FileReader();

    reader.onload = (e) => {
      try {
        const content = e.target?.result as string;
        const schema = JSON.parse(content);

        // Valider le schéma
        if (!schema.nodes) {
          alert('Format de fichier invalide');
          return;
        }

        // Charger les données
        this.nodes.set(schema.nodes);

        // Support ancien format (v1.0, v2.0) et nouveau format (v3.0)
        if (schema.version === '3.0' && schema.connections) {
          // Nouveau format v3.0: connexions unifiées
          this.connections.set(schema.connections);
        } else if (schema.version === '2.0' || schema.dataConnections || schema.flowConnections) {
          // Format v2.0: migrer vers liste unifiée
          const allConnections = [
            ...(schema.dataConnections || []),
            ...(schema.flowConnections || []),
          ];
          this.connections.set(allConnections);
        } else if (schema.connections) {
          // Ancien format v1.0: migrer les connexions simples
          this.connections.set(schema.connections);
        }
        if (schema.panOffset) {
          this.panOffset.set(schema.panOffset);
        }
        if (schema.nodeIdCounter !== undefined) {
          this.nodeIdCounter = schema.nodeIdCounter;
        }
        if (schema.connectionIdCounter !== undefined) {
          this.connectionIdCounter = schema.connectionIdCounter;
        }

        // Réinitialiser l'input pour permettre de charger le même fichier à nouveau
        input.value = '';
      } catch (error) {
        alert('Erreur lors du chargement du fichier: ' + error);
      }
    };

    reader.readAsText(file);
  }

  clearSchema() {
    if (confirm('Êtes-vous sûr de vouloir effacer tout le schéma ?')) {
      this.nodes.set([]);
      this.connections.set([]);
      this.panOffset.set({ x: 0, y: 0 });
      this.nodeIdCounter = 0;
      this.connectionIdCounter = 0;
    }
  }

  // Console integration
  // TODO : Faire en sorte que les logs viennent du ws d'interaction
  onJobExecute() {
    const console = this.bottomBar?.getConsole();
    if (!console) return;

    const nodeCount = this.nodes().length;
    const connectionCount = this.connections().length;

    console.addLog('info', `Analyse du schéma: ${nodeCount} nodes, ${connectionCount} connexions`);

    // Simulation d'exécution - à remplacer par vraie logique
    this.nodes().forEach((node, index) => {
      setTimeout(() => {
        this.bottomBar?.getConsole()?.addLog('info', `Traitement du node: ${node.type.label} (${node.id})`);
      }, (index + 1) * 500);
    });

    // Simulation de fin
    setTimeout(() => {
      this.bottomBar?.getConsole()?.markSuccess();
    }, (this.nodes().length + 1) * 500 + 500);
  }

  onJobStop() {
    this.bottomBar?.getConsole()?.addLog('warn', 'Exécution interrompue par l\'utilisateur');
  }
}
