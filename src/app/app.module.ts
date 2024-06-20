import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { NodeComponent } from './views/node/node.component';
import { NodeGraphComponent } from './views/node-graph/node-graph.component';
import { ContextMenuComponent } from './views/context-menu/context-menu.component';

@NgModule({
  declarations: [
    AppComponent,
    NodeComponent,
    NodeGraphComponent,
    ContextMenuComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule { }
