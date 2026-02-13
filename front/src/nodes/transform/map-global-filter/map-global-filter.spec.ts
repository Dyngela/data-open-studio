import { ComponentFixture, TestBed } from '@angular/core/testing';

import { MapGlobalFilter } from './map-global-filter';

describe('MapGlobalFilter', () => {
  let component: MapGlobalFilter;
  let fixture: ComponentFixture<MapGlobalFilter>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MapGlobalFilter]
    })
    .compileComponents();

    fixture = TestBed.createComponent(MapGlobalFilter);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
