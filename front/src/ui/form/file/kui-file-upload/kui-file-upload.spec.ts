import { ComponentFixture, TestBed } from '@angular/core/testing';

import { KuiFileUpload } from './kui-file-upload';

describe('KuiFileUpload', () => {
  let component: KuiFileUpload;
  let fixture: ComponentFixture<KuiFileUpload>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [KuiFileUpload]
    })
    .compileComponents();

    fixture = TestBed.createComponent(KuiFileUpload);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
