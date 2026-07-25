import { SPORTIUS_SERVICE } from '@sneat/extension-sportius-contract';
import { ListService } from './services';
import { provideSportius } from './provide-sportius';

describe('provideSportius', () => {
  it('provides ListService and binds it to SPORTIUS_SERVICE', () => {
    const providers = provideSportius();
    expect(providers).toContain(ListService);
    expect(providers).toContainEqual({
      provide: SPORTIUS_SERVICE,
      useExisting: ListService,
    });
  });
});
