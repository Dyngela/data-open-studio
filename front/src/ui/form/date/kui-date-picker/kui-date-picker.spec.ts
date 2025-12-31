import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiDatePicker } from './kui-date-picker';

describe('KuiDatePicker', () => {
  let component: KuiDatePicker;
  let fixture: ComponentFixture<KuiDatePicker>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiDatePicker]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiDatePicker);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
