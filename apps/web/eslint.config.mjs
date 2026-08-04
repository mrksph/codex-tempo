import { FlatCompat } from "@eslint/eslintrc";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dirname = path.dirname(fileURLToPath(import.meta.url));
const compat = new FlatCompat({ baseDirectory: dirname });
const config = [
  { ignores: [".next/**", "node_modules/**", "lib/api/generated.ts"] },
  ...compat.extends("next/core-web-vitals", "next/typescript"),
];
export default config;
