import { Component, input, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance } from '../../nodes/node.type';

interface MinimapBounds {
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
  width: number;
  height: number;
}

@Component({
  selector: 'app-minimap',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './minimap.html',
  styleUrl: './minimap.css',
})
export class Minimap {
  nodes = input<NodeInstance[]>([]);
  viewportWidth = input(800);
  viewportHeight = input(600);
  panOffset = input({ x: 0, y: 0 });

  minimapWidth = 200;
  minimapHeight = 150;

  bounds = computed(() => this.calculateBounds());
  scale = computed(() => this.calculateScale());
  transformedNodes = computed(() => this.transformNodes());

  private calculateBounds(): MinimapBounds | null {
    if (this.nodes().length === 0) {
      return null;
    }

    const nodeWidth = 180;
    const nodeHeight = 100;
    const padding = 50;

    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;

    for (const node of this.nodes()) {
      minX = Math.min(minX, node.position.x);
      minY = Math.min(minY, node.position.y);
      maxX = Math.max(maxX, node.position.x + nodeWidth);
      maxY = Math.max(maxY, node.position.y + nodeHeight);
    }

    minX = Math.max(0, minX - padding);
    minY = Math.max(0, minY - padding);
    maxX += padding;
    maxY += padding;

    return {
      minX,
      minY,
      maxX,
      maxY,
      width: maxX - minX,
      height: maxY - minY,
    };
  }

  private calculateScale(): number {
    const bounds = this.bounds();
    if (!bounds) return 1;

    const scaleX = this.minimapWidth / bounds.width;
    const scaleY = this.minimapHeight / bounds.height;

    return Math.min(scaleX, scaleY, 1);
  }

  private transformNodes(): Array<{ x: number; y: number; width: number; height: number; color: string }> {
    const bounds = this.bounds();
    const scale = this.scale();
    if (!bounds) return [];

    const nodeWidth = 180;
    const nodeHeight = 100;

    return this.nodes().map((node) => ({
      x: (node.position.x - bounds.minX) * scale,
      y: (node.position.y - bounds.minY) * scale,
      width: nodeWidth * scale,
      height: nodeHeight * scale,
      color: node.type.color || '#2196F3',
    }));
  }

  viewportRect = computed(() => {
    const bounds = this.bounds();
    const scale = this.scale();
    const offset = this.panOffset();
    if (!bounds) return null;

    return {
      x: (-offset.x - bounds.minX) * scale,
      y: (-offset.y - bounds.minY) * scale,
      width: this.viewportWidth() * scale,
      height: this.viewportHeight() * scale,
    };
  });
}
