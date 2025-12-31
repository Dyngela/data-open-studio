import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiMultiselect } from './kui-multiselect';

describe('KuiMultiselect', () => {
  let component: KuiMultiselect;
  let fixture: ComponentFixture<KuiMultiselect>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiMultiselect]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiMultiselect);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
