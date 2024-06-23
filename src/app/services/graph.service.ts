import {ElementRef, Injectable } from '@angular/core';
import {Graph} from "../models/graph.model";
import {NodeTypes} from "../enums/node.enum";

@Injectable({
  providedIn: 'root'
})
export class GraphService {

  private graph: Graph;
  private svgContainer!: ElementRef<SVGElement>;

  constructor() {
    this.graph = {
      nodes: [
        {
          id: '1',
          name: 'Start Node',
          inputs: [],
          outputs: [{ id: 'output1', type: 'output', connectedTo: ['2input2']}],
          position: { x: 100, y: 100 },
          type: NodeTypes.START
        },
        {
          id: '2',
          name: 'Connection Node',
          inputs: [{ id: '2input2', type: 'input', connectedTo: ['output1']}],
          outputs: [],
          position: { x: 500, y: 500 },
          type: NodeTypes.START
        }
      ],
    };
  }

  getSvgContainer() {
    return this.svgContainer;
  }

  setSvgContainer(svgContainer: ElementRef<SVGElement>) {
    this.svgContainer = svgContainer;
  }

  getGraph() {
    return this.graph;
  }

  setGraph(graph: any) {
    this.graph = graph;
  }
}
