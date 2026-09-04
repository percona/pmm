import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { AxiosError, AxiosResponse } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthProvider } from "./auth.provider";
import { wrapWithQueryProvider } from "utils/testUtils";

const { rotateToken, redirectToLogin, useFrontendSettings } = vi.hoisted(
  () => ({
    rotateToken: vi.fn(),
    redirectToLogin: vi.fn(),
    useFrontendSettings: vi.fn(),
  }),
);

vi.mock("api/auth", () => ({ rotateToken }));

vi.mock("./auth.utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./auth.utils")>()),
  redirectToLogin,
}));

vi.mock("hooks/api/useSettings", () => ({ useFrontendSettings }));

const httpError = (status: number) =>
  new AxiosError(
    `Request failed with status code ${status}`,
    String(status),
    undefined,
    undefined,
    { status } as AxiosResponse,
  );

const setSessionExpiry = (unixSeconds: number) => {
  document.cookie = `grafana_session_expiry=${unixSeconds}`;
};

const renderProvider = () =>
  render(
    wrapWithQueryProvider(
      <AuthProvider>
        <div>protected</div>
      </AuthProvider>,
    ),
  );

describe("AuthProvider", () => {
  beforeEach(() => {
    useFrontendSettings.mockReturnValue({ data: {}, isLoading: false });
  });

  afterEach(() => {
    localStorage.clear();
    document.cookie =
      "grafana_session_expiry=; expires=Thu, 01 Jan 1970 00:00:00 GMT";
    vi.clearAllMocks();
  });

  it("renders children once the token is rotated", async () => {
    rotateToken.mockResolvedValue({ message: "ok" });

    renderProvider();

    expect(await screen.findByText("protected")).toBeInTheDocument();
    expect(rotateToken).toHaveBeenCalledTimes(1);
    expect(redirectToLogin).not.toHaveBeenCalled();
  });

  it("redirects to login when the session cannot be rotated", async () => {
    rotateToken.mockRejectedValue(httpError(401));

    renderProvider();

    await waitFor(() => expect(redirectToLogin).toHaveBeenCalledTimes(1));
    expect(screen.queryByText("protected")).not.toBeInTheDocument();
    // this query gates the first paint, so a failure must not sit behind a backoff
    expect(rotateToken).toHaveBeenCalledTimes(1);
  });

  // Timers are frozen in hidden tabs, so the scheduled rotation can be missed outright.
  it("rotates on return to the tab when the deadline has passed", async () => {
    rotateToken.mockResolvedValue({ message: "ok" });

    renderProvider();
    await screen.findByText("protected");

    setSessionExpiry(Math.floor(Date.now() / 1000) - 60);
    // React Query's focus manager listens on `window`, which the real event reaches by
    // bubbling up from `document`.
    document.dispatchEvent(new Event("visibilitychange", { bubbles: true }));

    await waitFor(() => expect(rotateToken).toHaveBeenCalledTimes(2));
  });

  it("does not rotate on return to the tab while the session is still valid", async () => {
    rotateToken.mockResolvedValue({ message: "ok" });
    setSessionExpiry(Math.floor(Date.now() / 1000) + 3600);

    renderProvider();
    await screen.findByText("protected");

    // React Query's focus manager listens on `window`, which the real event reaches by
    // bubbling up from `document`.
    document.dispatchEvent(new Event("visibilitychange", { bubbles: true }));

    await waitFor(() => expect(rotateToken).toHaveBeenCalledTimes(1));
  });
});
