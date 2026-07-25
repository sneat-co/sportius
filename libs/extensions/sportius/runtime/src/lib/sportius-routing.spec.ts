import { sportiusRoutes } from './sportius-routing';

describe('sportiusRoutes', () => {
  it('exposes the lists overview route', () => {
    expect(sportiusRoutes.some((r) => r.path === 'lists')).toBe(true);
  });

  it('exposes the list detail route with listType + listID params', () => {
    expect(
      sportiusRoutes.some((r) => r.path === 'list/:listType/:listID'),
    ).toBe(true);
  });

  it('lazy-loads every route via loadComponent', () => {
    for (const route of sportiusRoutes) {
      expect(typeof route.loadComponent).toBe('function');
    }
  });
});
