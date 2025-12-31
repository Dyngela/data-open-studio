import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DbInput } from './db-input';

describe('DbInput', () => {
  let component: DbInput;
  let fixture: ComponentFixture<DbInput>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DbInput]
    })
    .compileComponents();

    fixture = TestBed.createComponent(DbInput);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
