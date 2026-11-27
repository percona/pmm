import { sendToSupportSchema } from './SendToSupport.schema';
import { SEND_TO_SUPPORT_DEFAULT_VALUES } from './SendToSupport.utils';

describe('sendToSupportSchema', () => {
  it('requires address, user, and password', () => {
    expect(
      sendToSupportSchema.safeParse(SEND_TO_SUPPORT_DEFAULT_VALUES).success
    ).toBe(false);
  });

  it('accepts the default address with credentials', () => {
    expect(
      sendToSupportSchema.safeParse({
        ...SEND_TO_SUPPORT_DEFAULT_VALUES,
        user: 'customer',
        password: 'secret',
      }).success
    ).toBe(true);
  });
});
