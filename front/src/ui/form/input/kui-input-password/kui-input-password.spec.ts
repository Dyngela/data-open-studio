import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiInputPassword } from './kui-input-password';

describe('KuiInputPassword', () => {
  let component: KuiInputPassword;
  let fixture: ComponentFixture<KuiInputPassword>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiInputPassword]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiInputPassword);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
