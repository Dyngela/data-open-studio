import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiGlobalLoadingSpinner } from './kui-global-loading-spinner';

describe('KuiGlobalLoadingSpinner', () => {
  let component: KuiGlobalLoadingSpinner;
  let fixture: ComponentFixture<KuiGlobalLoadingSpinner>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiGlobalLoadingSpinner]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiGlobalLoadingSpinner);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
