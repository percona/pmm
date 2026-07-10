import { sendToSupportSchema } from './SendToSupport.schema';
import {
  SEND_TO_SUPPORT_DEFAULT_VALUES,
  toUploadPayload,
} from './SendToSupport.utils';

describe('sendToSupportSchema', () => {
  it('requires address, user, and password', () => {
    expect(
      sendToSupportSchema.safeParse(SEND_TO_SUPPORT_DEFAULT_VALUES).success
    ).toBe(false);
  });

  it('maps valid form values to an upload payload', () => {
    const values = {
      ...SEND_TO_SUPPORT_DEFAULT_VALUES,
      user: 'customer',
      password: 'secret',
    };

    expect(toUploadPayload(values, ['dump-1'])).toEqual({
      dumpIds: ['dump-1'],
      sftpParameters: values,
    });
  });
});
