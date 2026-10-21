import { Component, For, createEffect, on, onCleanup, onMount } from "solid-js";
import { useLocation, useNavigate } from "@solidjs/router";
import Navbar from "@/components/Navbar";
import { MOBILE_TABS, indexFromPath, setPagerPosition } from "@/library/tabs";
import HomePage from "@/pages/HomePage/HomePage";
import { MyDrive } from "@/pages/MyDrive/MyDrive";
import MyCollections from "@/pages/MyCollections/MyCollections";
import Account from "@/pages/Account/Account";

/**
 * The four tab pages, in the same order as `MOBILE_TABS`.
 * All of them stay mounted for the whole session so that each page keeps its
 * own scroll position and component state while the user swipes between them.
 */
const PANES: ReadonlyArray<Component> = [HomePage, MyDrive, MyCollections, Account];

/**
 * Mobile tab pager.
 *
 * Renders the four tab pages side by side inside a native horizontal
 * scroll-snap container. Swiping reveals the real neighbouring page content,
 * releasing snaps to the nearest page, and the settled page is written back to
 * the URL (which in turn drives the navbar highlight).
 *
 * Native scroll-snap is used instead of a transform-based track so that we get
 * momentum, rubber-band edges, accessibility and per-pane scroll preservation
 * for free.
 */
const MobileTabShell: Component = () => {
    const location = useLocation();
    const navigate = useNavigate();

    let scroller: HTMLDivElement | undefined;
    let dragging = false;
    let settleTimer: number | undefined;
    // Target of an in-flight programmatic scroll (navbar tap / back button).
    // While set, scroll events must not be interpreted as a user swipe.
    let programmaticTarget: number | null = null;

    const paneWidth = () => (scroller ? scroller.clientWidth : window.innerWidth);
    const activeIndex = () => indexFromPath(location.pathname);

    // Publish the live fractional position so the navbar highlight can follow
    // the gesture. Clamped because iOS rubber-band can overshoot the range.
    const updatePosition = () => {
        if (!scroller) return;
        const width = paneWidth();
        if (width <= 0) return;
        const raw = scroller.scrollLeft / width;
        setPagerPosition(Math.max(0, Math.min(PANES.length - 1, raw)));
    };

    const scrollToIndex = (index: number, smooth: boolean) => {
        if (!scroller) return;
        const target = index * paneWidth();
        if (Math.abs(scroller.scrollLeft - target) < 1) {
            programmaticTarget = null;
            return;
        }
        programmaticTarget = target;
        scroller.scrollTo({ left: target, behavior: smooth ? "smooth" : "auto" });
        updatePosition();
    };

    // URL -> pager position (navbar taps, browser back/forward, deep links).
    createEffect(
        on(
            activeIndex,
            (index) => {
                scrollToIndex(index, true);
            },
            { defer: true }
        )
    );

    const scheduleSettle = () => {
        window.clearTimeout(settleTimer);
        settleTimer = window.setTimeout(settle, 120);
    };

    // Pager position -> URL (a swipe that has settled).
    const settle = () => {
        if (!scroller || dragging) return;
        const width = paneWidth();
        if (width <= 0) return;

        // Ignore intermediate positions while a programmatic scroll is running,
        // and keep polling until it reaches its target.
        if (programmaticTarget !== null) {
            if (Math.abs(scroller.scrollLeft - programmaticTarget) > 1) {
                scheduleSettle();
                return;
            }
            programmaticTarget = null;
        }

        const raw = Math.round(scroller.scrollLeft / width);
        const index = Math.max(0, Math.min(PANES.length - 1, raw));
        if (index !== activeIndex()) {
            navigate(MOBILE_TABS[index].route, { scroll: false });
        }
    };

    const handleScroll = () => {
        // Track the gesture continuously, even while the finger is down.
        updatePosition();
        // Momentum keeps rescheduling the timer, so this only fires once the
        // pager has actually come to rest.
        if (dragging) return;
        scheduleSettle();
    };

    const handleTouchStart = () => {
        dragging = true;
        programmaticTarget = null;
        window.clearTimeout(settleTimer);
    };

    const handleTouchEnd = () => {
        dragging = false;
        scheduleSettle();
    };

    const handleResize = () => {
        // Keep the pager aligned to the active tab without animating.
        scrollToIndex(activeIndex(), false);
        updatePosition();
    };

    onMount(() => {
        // Align immediately on first paint (deep link or reload). The pager may
        // not have been laid out yet, so retry on the next frame as well.
        scrollToIndex(activeIndex(), false);
        requestAnimationFrame(() => scrollToIndex(activeIndex(), false));
        setPagerPosition(activeIndex());
        window.addEventListener("resize", handleResize);
        window.addEventListener("orientationchange", handleResize);
    });

    onCleanup(() => {
        window.clearTimeout(settleTimer);
        window.removeEventListener("resize", handleResize);
        window.removeEventListener("orientationchange", handleResize);
    });

    return (
        <>
            <div
                ref={scroller}
                class="mobile-tab-pager flex w-screen h-dvh overflow-x-auto overflow-y-hidden snap-x snap-mandatory bg-black"
                onScroll={handleScroll}
                onTouchStart={handleTouchStart}
                onTouchEnd={handleTouchEnd}
                onTouchCancel={handleTouchEnd}
            >
                <For each={PANES}>
                    {(Pane, index) => (
                        <div
                            class="relative w-full h-full shrink-0 snap-start snap-always overflow-hidden"
                            // Off-screen panes stay mounted (to preserve their
                            // scroll position) but must not be reachable by
                            // keyboard or screen readers.
                            {...(index() !== activeIndex() ? { inert: true } : {})}
                            aria-hidden={index() !== activeIndex() ? "true" : undefined}
                        >
                            <Pane />
                        </div>
                    )}
                </For>
            </div>
            <Navbar Type="mobile" />
        </>
    );
};

export default MobileTabShell;
