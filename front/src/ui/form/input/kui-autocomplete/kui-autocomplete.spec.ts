import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiAutocomplete } from './kui-autocomplete';

describe('KuiAutocomplete', () => {
  let component: KuiAutocomplete;
  let fixture: ComponentFixture<KuiAutocomplete>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiAutocomplete]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiAutocomplete);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
