// @module: commonjs
// @esModuleInterop: true
// @declaration: true
// @strict: true

// @filename: graphics.d.ts
declare namespace graphics {
    interface Point { x: number; y: number }
}
declare var graphics: { Point: new (x: number, y: number) => graphics.Point };
declare module "graphics" {
    export = graphics;
}

// @filename: a.ts
import { Point as P1 } from "graphics";
import { Point as P2 } from "graphics";
import { Point as P3 } from "graphics";
declare const c: boolean;
declare const p1: P1[];
declare const p2: P2[];
declare const p3: P3[];
export const first = c ? p1 : p3;
export const made = new P2(0, 0);
export const u = c ? p1 : p2;
export const v = c ? p2 : p1;
