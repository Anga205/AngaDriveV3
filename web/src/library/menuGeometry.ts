/**
 * Pure geometry helpers for viewport-aware menus and the Menu Aim
 * ("safe triangle") algorithm. Kept free of DOM/Solid dependencies so the
 * math can be reasoned about (and unit tested) in isolation.
 */

export interface Point {
    x: number;
    y: number;
}

/** The subset of DOMRect that positioning needs. */
export interface RectLike {
    left: number;
    top: number;
    right: number;
    bottom: number;
    width: number;
    height: number;
}

export interface Size {
    width: number;
    height: number;
}

export interface ViewportBounds {
    left: number;
    top: number;
    right: number;
    bottom: number;
}

export type VerticalPlacement = "top" | "bottom";
export type HorizontalPlacement = "left" | "right";

export interface PanelPosition {
    left: number;
    top: number;
    /** Set when the panel is taller than the available space and must scroll. */
    maxHeight?: number;
    placement: VerticalPlacement;
}

export interface SubmenuPosition {
    left: number;
    top: number;
    maxHeight?: number;
    placement: HorizontalPlacement;
}

/** Clamp `value` into the inclusive range [min, max]. */
export function clamp(value: number, min: number, max: number): number {
    return Math.max(min, Math.min(max, value));
}

/** Usable viewport rectangle, inset by `margin` on every side. */
export function getViewportBounds(margin: number): ViewportBounds {
    return {
        left: margin,
        top: margin,
        right: window.innerWidth - margin,
        bottom: window.innerHeight - margin
    };
}

/**
 * Standard barycentric sign test. Returns true when `p` lies inside (or on
 * the edge of) the triangle a-b-c.
 */
export function pointInTriangle(p: Point, a: Point, b: Point, c: Point): boolean {
    const sign = (p1: Point, p2: Point, p3: Point) =>
        (p1.x - p3.x) * (p2.y - p3.y) - (p2.x - p3.x) * (p1.y - p3.y);

    const d1 = sign(p, a, b);
    const d2 = sign(p, b, c);
    const d3 = sign(p, c, a);

    const hasNegative = d1 < 0 || d2 < 0 || d3 < 0;
    const hasPositive = d1 > 0 || d2 > 0 || d3 > 0;
    return !(hasNegative && hasPositive);
}

/**
 * Compute fixed coordinates for a dropdown anchored to a trigger.
 *
 * Prefers opening below the anchor with the requested horizontal alignment,
 * flips above when there is more room there, and clamps horizontally so the
 * panel never leaves the viewport. If neither vertical side can fit the
 * panel, `maxHeight` is returned so the panel can scroll internally.
 */
export function computePanelPosition(
    anchorRect: RectLike,
    panelSize: Size,
    preferredAlign: "left" | "right",
    gap: number,
    margin: number
): PanelPosition {
    const bounds = getViewportBounds(margin);
    const { width, height } = panelSize;

    // Horizontal: align to the requested trigger edge, then clamp.
    let left = preferredAlign === "left" ? anchorRect.left : anchorRect.right - width;
    left = clamp(left, bounds.left, Math.max(bounds.left, bounds.right - width));

    // Vertical: prefer below, flip above when that side has more room.
    const spaceBelow = bounds.bottom - anchorRect.bottom - gap;
    const spaceAbove = anchorRect.top - gap - bounds.top;
    const placeBelow = spaceBelow >= height || spaceBelow >= spaceAbove;

    let top: number;
    let maxHeight: number | undefined;
    let placement: VerticalPlacement;

    if (placeBelow) {
        placement = "bottom";
        top = anchorRect.bottom + gap;
        if (height > spaceBelow) {
            maxHeight = Math.max(0, spaceBelow);
        }
    } else {
        placement = "top";
        top = anchorRect.top - gap - height;
        if (height > spaceAbove) {
            maxHeight = Math.max(0, spaceAbove);
            top = anchorRect.top - gap - maxHeight;
        }
    }

    top = Math.max(bounds.top, top);
    if (maxHeight !== undefined) {
        maxHeight = Math.max(0, Math.min(maxHeight, bounds.bottom - top));
    }

    return { left, top, maxHeight, placement };
}

/**
 * Compute fixed coordinates for a submenu anchored to its parent item.
 *
 * Prefers opening to the right, flips left near the right edge, aligns the
 * top with the parent item, and clamps vertically into the viewport.
 */
export function computeSubmenuPosition(
    itemRect: RectLike,
    panelSize: Size,
    gap: number,
    margin: number
): SubmenuPosition {
    const bounds = getViewportBounds(margin);
    const { width, height } = panelSize;

    const spaceRight = bounds.right - itemRect.right - gap;
    const spaceLeft = itemRect.left - gap - bounds.left;
    const placeRight = spaceRight >= width || spaceRight >= spaceLeft;

    let left: number;
    let placement: HorizontalPlacement;

    if (placeRight) {
        placement = "right";
        left = itemRect.right + gap;
        if (width > spaceRight) {
            left = Math.min(left, Math.max(bounds.left, bounds.right - width));
        }
    } else {
        placement = "left";
        left = itemRect.left - gap - width;
        if (width > spaceLeft) {
            left = Math.max(bounds.left, left);
        }
    }

    let top = clamp(itemRect.top, bounds.top, Math.max(bounds.top, bounds.bottom - height));
    let maxHeight: number | undefined;
    if (height > bounds.bottom - bounds.top) {
        maxHeight = Math.max(0, bounds.bottom - bounds.top);
        top = bounds.top;
    }

    return { left, top, maxHeight, placement };
}

/**
 * Build the Menu Aim "safe triangle".
 *
 * Vertices are the point where the pointer left the submenu trigger, the
 * near corner of the submenu panel, and a look-ahead point extrapolated from
 * the pointer's velocity. While the pointer remains inside this triangle (or
 * inside the panel itself) the submenu is considered "aimed at" and stays
 * open, even if the pointer briefly left the trigger.
 */
export function buildAimTriangle(
    exitPoint: Point,
    velocity: Point,
    nearCorner: Point,
    lookAheadMs: number
): [Point, Point, Point] {
    const projected: Point = {
        x: exitPoint.x + velocity.x * lookAheadMs,
        y: exitPoint.y + velocity.y * lookAheadMs
    };
    return [exitPoint, projected, nearCorner];
}
