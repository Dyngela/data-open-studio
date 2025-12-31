import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiLocalLoadingSpinner } from './kui-local-loading-spinner';

describe('KuiLocalLoadingSpinner', () => {
  let component: KuiLocalLoadingSpinner;
  let fixture: ComponentFixture<KuiLocalLoadingSpinner>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiLocalLoadingSpinner]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiLocalLoadingSpinner);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
