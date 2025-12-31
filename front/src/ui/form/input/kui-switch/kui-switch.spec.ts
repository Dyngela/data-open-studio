import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiSwitch } from './kui-switch';

describe('KuiSwitch', () => {
  let component: KuiSwitch;
  let fixture: ComponentFixture<KuiSwitch>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiSwitch]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiSwitch);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
