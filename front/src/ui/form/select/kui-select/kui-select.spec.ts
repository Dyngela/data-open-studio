import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiSelect } from './kui-select';

describe('KuiSelect', () => {
  let component: KuiSelect;
  let fixture: ComponentFixture<KuiSelect>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiSelect]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiSelect);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
