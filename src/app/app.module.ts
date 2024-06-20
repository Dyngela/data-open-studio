import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { NodeGraphComponent } from './views/node-graph/node-graph.component';
import { ContextMenuComponent } from './views/context-menu/context-menu.component';
import { StartNodeComponent } from './views/nodes/start-node/start-node.component';
import { DbConnNodeComponent } from './views/nodes/db-conn-node/db-conn-node.component';

@NgModule({
  declarations: [
    AppComponent,
    NodeGraphComponent,
    ContextMenuComponent,
    StartNodeComponent,
    DbConnNodeComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule { }
