import { NgModule } from '@angular/core';
import {
  ISportiusAppStateService,
  SportiusAppStateService,
} from './sportius-app-state.service';

// Provides the template UI-state service. The concrete ListService is no longer
// provided here — it is bound to the SPORTIUS_SERVICE contract token by
// provideSportius() at app bootstrap (the app is the composition root).
@NgModule({
  providers: [
    {
      provide: ISportiusAppStateService,
      useClass: SportiusAppStateService,
    },
  ],
})
export class SportiusCoreServicesModule {}
