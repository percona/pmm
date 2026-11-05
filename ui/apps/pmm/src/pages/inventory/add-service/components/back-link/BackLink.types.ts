// Either links to `to` or is handled in place by `onClick`.
export type BackLinkProps = { label: string } & (
  | { to: string; onClick?: never }
  | { onClick: () => void; to?: never }
);
