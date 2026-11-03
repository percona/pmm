import {
  mockAddService,
  mockNodes,
  renderServiceForm,
  sentPayload,
  submit,
  type,
} from '../ServiceForm.testUtils';
import { ValkeyForm } from './ValkeyForm';

vi.mock('api/services', () => ({ addService: vi.fn() }));
vi.mock('hooks/api/useNodes', () => ({ useNodes: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

describe('ValkeyForm', () => {
  beforeEach(() => {
    mockAddService();
    mockNodes();
  });

  it('adds a Valkey service without credentials', async () => {
    renderServiceForm(ValkeyForm);

    type('text-input-address', 'valkey.example.com');
    submit();

    expect(await sentPayload('valkey')).toEqual(
      expect.objectContaining({
        address: 'valkey.example.com',
        port: 6379,
        username: '',
        password: '',
        tls: false,
      })
    );
  });
});
