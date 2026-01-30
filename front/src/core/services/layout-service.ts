import {computed, inject, Injectable, model, signal} from '@angular/core';
import {NodeGraphService} from '../nodes-services/node-graph.service';

@Injectable({
  providedIn: 'root',
})
export class LayoutService {
  private nodeGraph = inject(NodeGraphService);

  // Bottom bar state
  public height = signal(200);
  public activeTab = signal<string>('console');

  public activeModal = signal<{ nodeId: number; nodeTypeId: string } | null>(null);
  viewportWidth = signal(0);
  viewportHeight = signal(0);

  // Calculer si un panneau latéral est ouvert
  isAnySidePanelOpen = computed(() => this.leftTabs().some(t => t.active && t.label !== 'Console'));
  IsBottomPanelOpen = computed(() => this.leftTabs().some(t => t.active && t.label === 'Console'));

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

  sidebarWidth = signal(250);
  isResizing = signal(false);

  openNodeModal(nodeId: number) {
    const node = this.nodeGraph.getNodeById(nodeId);
    if (!node) return;

    this.activeModal.set({ nodeId: node.id, nodeTypeId: node.type.id });
  }

  closeModal() {
    this.activeModal.set(null);
  }

  startResizing(e: MouseEvent) {
    this.isResizing.set(true);
    e.preventDefault();
  }
}
