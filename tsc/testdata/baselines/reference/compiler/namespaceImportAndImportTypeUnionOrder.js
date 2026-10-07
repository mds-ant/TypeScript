//// [tests/cases/compiler/namespaceImportAndImportTypeUnionOrder.ts] ////

//// [defaults.ts]
export const retries = 3;
export default { retries };

//// [use0.ts]
import * as defaults from "./defaults";
export function getConfig(override?: typeof import("./defaults")) {
    return override ?? defaults;
}

//// [use1.ts]
import * as defaults from "./defaults";
export function getConfig(override?: typeof import("./defaults")) {
    return override ?? defaults;
}

//// [use2.ts]
import * as defaults from "./defaults";
export function getConfig(override?: typeof import("./defaults")) {
    return override ?? defaults;
}

//// [use3.ts]
import * as defaults from "./defaults";
export function getConfig(override?: typeof import("./defaults")) {
    return override ?? defaults;
}


//// [defaults.js]
export const retries = 3;
export default { retries };
//// [use0.js]
import * as defaults from "./defaults";
export function getConfig(override) {
    return override ?? defaults;
}
//// [use1.js]
import * as defaults from "./defaults";
export function getConfig(override) {
    return override ?? defaults;
}
//// [use2.js]
import * as defaults from "./defaults";
export function getConfig(override) {
    return override ?? defaults;
}
//// [use3.js]
import * as defaults from "./defaults";
export function getConfig(override) {
    return override ?? defaults;
}


//// [defaults.d.ts]
export declare const retries = 3;
declare const _default: {
    retries: number;
};
export default _default;
//// [use0.d.ts]
export declare function getConfig(override?: typeof import("./defaults")): typeof import("./defaults");
//// [use1.d.ts]
export declare function getConfig(override?: typeof import("./defaults")): typeof import("./defaults");
//// [use2.d.ts]
export declare function getConfig(override?: typeof import("./defaults")): typeof import("./defaults");
//// [use3.d.ts]
export declare function getConfig(override?: typeof import("./defaults")): typeof import("./defaults");
