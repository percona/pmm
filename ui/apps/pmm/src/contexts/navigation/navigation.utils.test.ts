import { OrgRole, User } from 'types/user.types';
import { addAlerting } from './navigation.utils';

const baseUser = (overrides: Partial<User> = {}): User =>
  ({
    id: 1,
    name: 'Test',
    login: 'test',
    isAnonymous: false,
    orgId: 1,
    orgRole: OrgRole.Viewer,
    isAuthorized: true,
    isViewer: true,
    isEditor: false,
    isPMMAdmin: false,
    orgs: [],
    info: {} as User['info'],
    preferences: {} as User['preferences'],
    ...overrides,
  }) as User;

const childIds = (item: ReturnType<typeof addAlerting>) =>
  (item.children || []).map((c) => c.id);

describe('addAlerting', () => {
  it('always includes alert rules', () => {
    expect(childIds(addAlerting())).toContain('alerts-rules');
  });

  it('does not add user-gated pages without a user', () => {
    const ids = childIds(addAlerting(true, true));

    expect(ids).toContain('alerts-rules');
    expect(ids).toContain('alerts-status');
    expect(ids).not.toContain('alerts-silences');
    expect(ids).not.toContain('alerts-groups');
    expect(ids).not.toContain('alerts-contact-points');
    expect(ids).not.toContain('alerts-policies');
    expect(ids).not.toContain('alerts-settings');
  });

  it('adds silences for anonymous when UA is off, without groups or UA pages', () => {
    const ids = childIds(
      addAlerting(false, false, baseUser({ isAnonymous: true }))
    );

    expect(ids).toContain('alerts-rules');
    expect(ids).toContain('alerts-silences');
    expect(ids).not.toContain('alerts-groups');
    expect(ids).not.toContain('alerts-contact-points');
    expect(ids).not.toContain('alerts-policies');
    expect(ids).not.toContain('alerts-settings');
  });

  it('adds silences, contact points and policies for anonymous when UA is on', () => {
    const ids = childIds(
      addAlerting(false, true, baseUser({ isAnonymous: true }))
    );

    expect(ids).toContain('alerts-rules');
    expect(ids).toContain('alerts-silences');
    expect(ids).toContain('alerts-contact-points');
    expect(ids).toContain('alerts-policies');
    expect(ids).not.toContain('alerts-groups');
    expect(ids).not.toContain('alerts-settings');
  });

  it('adds silences and groups for authenticated user when UA is off', () => {
    const ids = childIds(addAlerting(false, false, baseUser()));

    expect(ids).toContain('alerts-silences');
    expect(ids).toContain('alerts-groups');
    expect(ids).not.toContain('alerts-contact-points');
    expect(ids).not.toContain('alerts-policies');
    expect(ids).not.toContain('alerts-settings');
  });

  it('adds contact points and policies for authenticated user when UA is on', () => {
    const ids = childIds(addAlerting(false, true, baseUser()));

    expect(ids).toContain('alerts-silences');
    expect(ids).toContain('alerts-groups');
    expect(ids).toContain('alerts-contact-points');
    expect(ids).toContain('alerts-policies');
  });

  it('adds alert settings only for non-anonymous PMM admin when UA is on', () => {
    expect(
      childIds(addAlerting(false, true, baseUser({ isPMMAdmin: true })))
    ).toContain('alerts-settings');
    expect(
      childIds(addAlerting(false, true, baseUser({ isPMMAdmin: false })))
    ).not.toContain('alerts-settings');
    expect(
      childIds(addAlerting(false, false, baseUser({ isPMMAdmin: true })))
    ).not.toContain('alerts-settings');
    expect(
      childIds(
        addAlerting(
          false,
          true,
          baseUser({ isAnonymous: true, isPMMAdmin: true })
        )
      )
    ).not.toContain('alerts-settings');
  });

  it('adds status when alerting is enabled and templates only for editors', () => {
    expect(childIds(addAlerting(true, false))).toContain('alerts-status');
    expect(childIds(addAlerting(false, false))).not.toContain('alerts-status');

    expect(
      childIds(addAlerting(true, false, baseUser({ isEditor: true })))
    ).toContain('alerts-templates');
    expect(
      childIds(addAlerting(true, false, baseUser({ isEditor: false })))
    ).not.toContain('alerts-templates');
    expect(
      childIds(addAlerting(false, false, baseUser({ isEditor: true })))
    ).not.toContain('alerts-templates');
  });
});
