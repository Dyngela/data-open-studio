import { ComponentFixture, TestBed } from '@angular/core/testing';

import { StartNode } from './start-node';

describe('StartNode', () => {
  let component: StartNode;
  let fixture: ComponentFixture<StartNode>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StartNode]
    })
    .compileComponents();

    fixture = TestBed.createComponent(StartNode);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
