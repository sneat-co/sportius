import { Route } from '@angular/router';
import { SpaceComponentBaseParams } from '@sneat/space-components';
import { SportiusSpaceMenuComponent } from './space-menu/sportius-space-menu.component';
import { sportiusRoutes } from './sportius-routing';

export const sportiusSpaceRoutes: Route[] = [
  {
    path: '',
    providers: [SpaceComponentBaseParams],
    children: [
      {
        path: '',
        component: SportiusSpaceMenuComponent,
        outlet: 'menu',
      },
      {
        path: '',
        pathMatch: 'full',
        redirectTo: 'lists',
      },
      ...sportiusRoutes,
    ],
  },
];
