import { ComponentFixture, TestBed } from '@angular/core/testing';

import { MapOutputField } from './map-output-field';

describe('MapOutputField', () => {
  let component: MapOutputField;
  let fixture: ComponentFixture<MapOutputField>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MapOutputField]
    })
    .compileComponents();

    fixture = TestBed.createComponent(MapOutputField);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
