import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiSubmitButton } from './kui-submit-button';

describe('KuiSubmitButton', () => {
  let component: KuiSubmitButton;
  let fixture: ComponentFixture<KuiSubmitButton>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiSubmitButton]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiSubmitButton);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
