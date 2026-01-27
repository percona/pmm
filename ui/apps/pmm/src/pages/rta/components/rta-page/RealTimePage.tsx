import Stack from "@mui/material/Stack"
import { FC, PropsWithChildren } from "react"

const RealTimePage: FC<PropsWithChildren> = ({ children }) => (
    <Stack
        direction="column"
        gap={2}
        p={2}
        sx={{
            height: '100%',
            maxHeight: 'calc(100vh - 64px)', // Account for header height
            overflow: 'hidden',
            display: 'flex',
        }}>
        {children}
    </Stack>
)

export default RealTimePage;