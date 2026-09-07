# Transactional emails

Help the recipient complete one account action: create an account, reset a password, or confirm a new email address. Each message names the action, explains the next step, gives the actual link lifetime, and explains what to do with an unsolicited request.

## Visual contract

Agora's world-building identity uses a quiet blue-black field, a borderless content island, and a cyan primary action. The text masthead remains readable with images disabled. The narrative signoff borrows “Build worlds worth returning to.” from uikit's typography specimen; the French adaptation keeps its meaning and the existing informal voice.

The shared theme is an email-compatible snapshot of [uikit's foundations](https://github.com/a-novel-kit/uikit/tree/4359176d8e4cb88180565f9c71c5176a03976c9e/packages/tokens). Colors are measured in Chromium's sRGB canvas. MJML emits literal values and inline styles so clients can render the design without CSS custom properties, OKLCH, or web fonts.

| Email role      | Uikit token              | sRGB value |
| --------------- | ------------------------ | ---------- |
| Canvas          | `--color-surface-canvas` | `#010203`  |
| Content island  | `--color-surface-raised` | `#161a1d`  |
| Heading         | `--color-text-primary`   | `#edf3f6`  |
| Body            | `--color-text-secondary` | `#c5cbcf`  |
| Supporting text | `--color-text-muted`     | `#a2a8ab`  |
| Action          | `--color-action-primary` | `#0098db`  |
| Action text     | `--color-text-inverse`   | `#030506`  |
| Link and focus  | `--color-text-link`      | `#00b1ff`  |

Type uses uikit's display, interface, monospace, and story roles with locally available email font fallbacks. Body text is 16px with 1.5 line height; headings round the 4xl step to 29px. Spacing follows the 4px base, with 8px action and 12px island radii. A 600px fluid layout reduces content padding on narrow screens and lets long French labels wrap. The primary action has at least 48px of height.

Copy follows [uikit's content guidance](https://github.com/a-novel-kit/uikit/blob/4359176d8e4cb88180565f9c71c5176a03976c9e/packages/storybook/src/guidelines/ContentLayout.mdx): sentence case, verb-led actions, consistent vocabulary, and specific next steps. Expiry is factual. The password and email-change messages explain that ignoring the request leaves the current credential unchanged.

## Editing and review

Edit the localized `.mjml` sources and shared `mj-theme.mjml` / `mj-header.mjml` partials, then run `pnpm generate:mjml`. Commit the six generated HTML files with their sources. `pnpm lint:mjml` validates every complete message, including its partials, without writing files.

`Emails.mdx` is a portable Storybook docs page for the generated HTML. Add it to a Vite-based Storybook workbench's `stories` list with the docs addon enabled. It renders all message and language combinations with sample values on a reserved example domain. The preview iframe is sandboxed and can switch between desktop and narrow widths.

Check all six messages at 320px and desktop widths, keyboard focus, long URLs, increased text size, and contrast. The Go rendering test checks MIME headers, language, lifetime, and matching primary/fallback action URLs with their complete query parameters. Browser previews validate layout; inbox testing remains necessary for client-specific transformations in Gmail, Apple Mail, and desktop Outlook.

## Reference

[Privacy's confirmation email](https://reallygoodemails.com/emails/please-confirm-your-email-2) supplied a reference for an explicit confirmation subject. Agora's visual values and wording come from uikit and the service's existing account flows.
