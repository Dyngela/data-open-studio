import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Script } from './script';

describe('Script', () => {
  let component: Script;
  let fixture: ComponentFixture<Script>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Script]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Script);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
