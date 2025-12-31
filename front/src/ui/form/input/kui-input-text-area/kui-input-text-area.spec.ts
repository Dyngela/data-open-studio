import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiInputTextArea } from './kui-input-text-area';

describe('KuiInputTextArea', () => {
  let component: KuiInputTextArea;
  let fixture: ComponentFixture<KuiInputTextArea>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiInputTextArea]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiInputTextArea);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
