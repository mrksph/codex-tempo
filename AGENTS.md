# Agent Notes

- Tailwind CSS is the stylesheet strategy for the frontend. Shadcn UI and daisyUI are not used.
- Prefer utility-first styling in components (Tailwind classes) instead of adding feature-specific rules in `apps/web/app/globals.css`.
- Avoid adding new feature styling directly to `apps/web/app/globals.css` unless it is truly global or cannot be expressed cleanly elsewhere.
- If a component library is introduced later, prefer it over ad hoc custom CSS for reusable UI.
- Keep UI changes aligned with the existing design system and avoid widening the CSS surface without a clear reason.
