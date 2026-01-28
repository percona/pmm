import { render, screen, waitFor } from "@testing-library/react";
import RealtimeTab from "./RealtimeTab";
import { wrapWithQueryProvider } from "utils/testUtils";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { TEST_REAL_TIME_SESSION } from "utils/testStubs";

const { getRunningSessions } = vi.hoisted(() => ({
    getRunningSessions: vi.fn().mockResolvedValue([]),
}));

vi.mock('api/rta', () => ({
    getRunningSessions,
}));


const renderComponent = () => render(
    wrapWithQueryProvider(
        <MemoryRouter>
            <Routes>
                <Route path="/" element={<RealtimeTab />} />
                <Route path="/rta/selection" element={<div data-testid="realtime-selection">Selection</div>} />
                <Route path="/rta/sessions" element={<div data-testid="realtime-sessions">Sessions</div>} />
            </Routes>
        </MemoryRouter>
    ));

describe('RealtimeTab', () => {
    beforeEach(() => {
        getRunningSessions.mockClear();
    });

    it('should render loading', () => {
        renderComponent();

        expect(screen.getByTestId('realtime-tab-loading')).toBeInTheDocument();
    });

    it('should navigate to selection page if no sessions are running', async () => {
        renderComponent();

        await waitFor(() => {
            expect(screen.getByTestId('realtime-selection')).toBeInTheDocument();
        });
    });

    it('should navigate to sessions page if sessions are running', async () => {
        getRunningSessions.mockResolvedValue([TEST_REAL_TIME_SESSION]);

        renderComponent();

        await waitFor(() => {
            expect(screen.getByTestId('realtime-sessions')).toBeInTheDocument();
        });
    });
});