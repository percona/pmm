# Peak UI — guide for AI coding agents

This guide explains how to use Peak UI correctly inside an app that depends on it. It contains rules and boundaries only; the component catalog lives in Storybook.

Working on Peak UI itself (the `percona/peak-ui` repository), i.e., contributing to it? Then read the `CONTRIBUTING.md` file instead.

## What Peak UI is

`@percona/peak-ui` is Percona's React component library built on top of **MUI**. It ships:

- a Percona theme (light + dark) with design tokens,
- a theme wrapper every app must mount once,
- form inputs pre-wired to **react-hook-form**,
- a small set of composite components that add behavior on top of MUI.

Storybook is the definitive, coded source of truth: **https://percona.github.io/peak-ui/**. Search it before building anything.

## Where Peak UI ends and MUI begins

- **Peak UI does not re-export MUI.** Buttons, layout (`Box`, `Stack`, `Grid`), `Typography`, menus, alerts, etc. come from `@mui/material`; icons from `@mui/icons-material`. Use path imports: `import Button from '@mui/material/Button'`.
- **Peak UI exports only what adds value over MUI.** Before writing a component, check whether `@percona/peak-ui` already exports it. If it does, use it. If not, use MUI directly — the Peak UI theme already styles every MUI component to "look like Percona".
- **Never hand-style MUI to "look like Percona".** No hex colors, no custom fonts, no border radius tweaks. If something looks off, the theme is wrong or missing; do not patch it locally; ask the user what to do or help them file an issue against Peak UI.
- **Import from the package root only:** `import { TextInput } from '@percona/peak-ui'`. Never deep-import from `@percona/peak-ui/dist/...`. The package is published as ESM only (`"type": "module"`); there is no CommonJS build.
- **Customize through the exposed surface.** Peak UI components expose dedicated `*Props` slot props for the MUI component they wrap (for example `textFieldProps`), and some also accept `sx`; many expose only the slots, so check the props type. Do not target internal class names or wrap a Peak UI component just to restyle it.
- **Respect the Storybook maturity tag** on each component: `stable` — build on it freely; `experimental` — API may change; `needs-review` — use with care; `deprecated` — do not use for new work.

## Theme wrapper (required, exactly once)

```tsx
import { ThemeContextProvider, pmmThemeOptions } from '@percona/peak-ui';

export const Root = () => (
  <ThemeContextProvider themeOptions={pmmThemeOptions} saveColorModeOnLocalStorage>
    <App />
  </ThemeContextProvider>
);
```

- `ThemeContextProvider` creates the MUI theme, renders `CssBaseline`, and manages light/dark mode. **Do not add your own `ThemeProvider`, `createTheme`, or `CssBaseline`** on top of it.
- Pick one theme option: `baseThemeOptions` (Percona default), `pmmThemeOptions` (Percona Monitoring and Management), `sepThemeOptions` (SEP). Each is a function of the palette mode, `(mode: 'light' | 'dark') => ThemeOptions`; pass the function itself, do not call it.
- Toggle or read the color mode via `ColorModeContext`: `const { colorMode, toggleColorMode } = useContext(ColorModeContext)`. `saveColorModeOnLocalStorage` persists the choice.
- Read design values from the theme (`useTheme()`, or `sx={{ color: 'text.secondary', p: 2 }}`), never from hard-coded literals. Spacing is in theme units (`gap: 2` = 16px).
- Avoid overriding the theme in the app; missing tokens or variants belong in Peak UI (see above).
- **Fonts are not loaded for you.** The theme names Poppins (weights 400, 500, 600, 700) and Roboto Mono (weight 450, so use the variable font). Load them once at your app entry: install the font packages yourself (e.g. `@fontsource/poppins` and `@fontsource-variable/roboto-mono`) or add a Google Fonts `<link>`. Do not import Peak UI's own `@fontsource/*` dependencies; they are not part of its public contract and resolve only on hoisted installs.
- **Date/time inputs** need MUI X's `LocalizationProvider` with a date adapter (e.g. `AdapterDateFnsV3`) above them. Peak UI does not provide it.
- **Snackbars:** notistack is a peer dependency. Register `NotistackMuiSnackbar` as the notistack `Components` renderer so toasts use MUI `Alert`.

## Form inputs and react-hook-form

Every form input exported by Peak UI is a react-hook-form `Controller` under the hood.

```tsx
import { FormProvider, useForm } from 'react-hook-form';
import { TextInput } from '@percona/peak-ui';

type Values = { host: string };

const methods = useForm<Values>({ defaultValues: { host: '' } });

<FormProvider {...methods}>
  <TextInput<Values>
    name="host"
    label="Host"
    isRequired
    controllerProps={{ rules: { required: 'Host is required' } }}
    textFieldProps={{ placeholder: 'db.example.com' }}
  />
</FormProvider>
```

- `name` is required and is the react-hook-form field path. Wrap inputs in `FormProvider` (preferred) or pass `control` explicitly. Never render an input outside both.
- **Do not pass `value`, `onChange`, or `defaultValue`.** The Controller owns them; set defaults in `useForm({ defaultValues })`.
- **Validation belongs to react-hook-form:** `controllerProps={{ rules }}` or a resolver (zod, yup). Errors render automatically as helper text. `isRequired` only adds the asterisk and `required` attribute; it does not validate.
- Props for the wrapped MUI component go through its slot prop (`textFieldProps`, `selectFieldProps`, `slotProps`, ...), never spread onto the Peak UI component.
- Inputs are generic over your form values; type them (`<TextInput<Values> name="host" />`) so `name` is checked.
- Inputs render a `data-testid` built from the kebab-cased `name` with a prefix that varies per input (for example `text-input-host`); `RadioGroup` uses the option value instead. Read the exact id in Storybook or the DOM and reuse it rather than adding your own.

## Naming and layout conventions

- **Polymorphic prop is `component`, not `as`:** `<Typography component="h1">`, `<Button component={RouterLink} to="/x">`.
- **Space siblings with `gap`, not `spacing`:** `sx={{ display: 'flex', gap: 2 }}` on the parent rather than `Stack spacing`, `Grid spacing`, or margins on children. (`PageContainer` is the exception: it is a `Stack` and exposes `spacing`.)
- Color and typography come from theme keys (`'primary.main'`, `variant="body2"`), not from CSS literals.
- Peak UI components use named exports; import them from `@percona/peak-ui`.

## Peer dependencies

Peak UI does not bundle React, MUI, Emotion, notistack, or react-hook-form; the app installs them. The install command lives in the package `README.md` (also in `node_modules/@percona/peak-ui`) and the authoritative list is `peerDependencies` in `package.json`. Add a date adapter (e.g. `date-fns`) if you use date/time inputs.

## Links

- Storybook (components, tokens, usage): https://percona.github.io/peak-ui/
- Source and issues: https://github.com/percona/peak-ui
- Figma kit (design intent): https://www.figma.com/design/08jGF3GZAUGmazlQtk0UQk/Peak-Design-Kit
- MUI docs: https://mui.com/material-ui/
