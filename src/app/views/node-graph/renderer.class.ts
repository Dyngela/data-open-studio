import {Component} from "@angular/core";
import {Node} from "../../models/node.model";

@Component({
  template: '' // This will be an abstract base class, so no template
})
export class RendererClass {

  connections: { from: Node; to: Node; fromConnectorId: string, toConnectorId: string, path: string }[] = [];


  constructor() {}

}
