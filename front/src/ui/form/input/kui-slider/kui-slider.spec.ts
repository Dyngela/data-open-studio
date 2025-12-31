import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiSlider } from './kui-slider';

describe('KuiSlider', () => {
  let component: KuiSlider;
  let fixture: ComponentFixture<KuiSlider>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiSlider]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiSlider);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
