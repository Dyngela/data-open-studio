import { Injectable } from '@angular/core';
import {Graph} from "../models/graph.model";

@Injectable({
  providedIn: 'root'
})
export class GraphService {

  private graph: Graph;

  constructor() {
    this.graph = {
      nodes: [],
    };
  }

  getGraph() {
    return this.graph;
  }

  setGraph(graph: any) {
    this.graph = graph;
  }
}
