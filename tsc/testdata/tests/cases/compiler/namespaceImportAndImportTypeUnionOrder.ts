// @module: esnext
// @moduleResolution: bundler
// @declaration: true
// @strict: true

// @filename: defaults.ts
export const retries = 3;
export default { retries };

// @filename: use0.ts
import * as defaults from "./defaults";
export function getConfig(override?: typeof import("./defaults")) {
    return override ?? defaults;
}

// @filename: use1.ts
import * as defaults from "./defaults";
export function getConfig(override?: typeof import("./defaults")) {
    return override ?? defaults;
}

// @filename: use2.ts
import * as defaults from "./defaults";
export function getConfig(override?: typeof import("./defaults")) {
    return override ?? defaults;
}

// @filename: use3.ts
import * as defaults from "./defaults";
export function getConfig(override?: typeof import("./defaults")) {
    return override ?? defaults;
}
