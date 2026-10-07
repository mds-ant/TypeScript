//// [tests/cases/compiler/importedValueAndTypeSymbolsOrder.ts] ////

//// [graphics.d.ts]
declare namespace graphics {
    interface Point { x: number; y: number }
}
declare var graphics: { Point: new (x: number, y: number) => graphics.Point };
declare module "graphics" {
    export = graphics;
}

//// [a.ts]
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


//// [a.js]
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.v = exports.u = exports.made = exports.first = void 0;
const graphics_1 = require("graphics");
exports.first = c ? p1 : p3;
exports.made = new graphics_1.Point(0, 0);
exports.u = c ? p1 : p2;
exports.v = c ? p2 : p1;


//// [a.d.ts]
import { Point as P1 } from "graphics";
import { Point as P3 } from "graphics";
export declare const first: P3[];
export declare const made: graphics.Point;
export declare const u: P1[];
export declare const v: P1[];
