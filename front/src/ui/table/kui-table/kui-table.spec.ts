import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiTable } from './kui-table';

describe('KuiTable', () => {
  let component: KuiTable;
  let fixture: ComponentFixture<KuiTable>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiTable]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiTable);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
