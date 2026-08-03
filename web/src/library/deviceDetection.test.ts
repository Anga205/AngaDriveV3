import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { isMobileDevice } from './deviceDetection';

describe('isMobileDevice', () => {
  const originalNavigator = { ...globalThis.navigator };

  beforeEach(() => {
    // Reset to a clean desktop state before each test
    Object.defineProperty(globalThis.navigator, 'userAgent', {
      value: 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0',
      configurable: true,
    });
    Object.defineProperty(globalThis.navigator, 'maxTouchPoints', {
      value: 0,
      configurable: true,
    });
    Object.defineProperty(globalThis.navigator, 'hardwareConcurrency', {
      value: 8,
      configurable: true,
    });
    // @ts-ignore
    delete (globalThis.window as any).ontouchstart;
  });

  afterEach(() => {
    Object.defineProperty(globalThis.navigator, 'userAgent', {
      value: originalNavigator.userAgent,
      configurable: true,
    });
    Object.defineProperty(globalThis.navigator, 'maxTouchPoints', {
      value: originalNavigator.maxTouchPoints,
      configurable: true,
    });
    Object.defineProperty(globalThis.navigator, 'hardwareConcurrency', {
      value: originalNavigator.hardwareConcurrency,
      configurable: true,
    });
  });

  it('returns false for desktop Linux Chrome', () => {
    expect(isMobileDevice()).toBe(false);
  });

  it('returns true for Android user agent', () => {
    Object.defineProperty(globalThis.navigator, 'userAgent', {
      value: 'Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/120.0.0.0 Mobile Safari/537.36',
      configurable: true,
    });
    expect(isMobileDevice()).toBe(true);
  });

  it('returns true for iPhone user agent', () => {
    Object.defineProperty(globalThis.navigator, 'userAgent', {
      value: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148',
      configurable: true,
    });
    expect(isMobileDevice()).toBe(true);
  });

  it('returns true for iPad user agent', () => {
    Object.defineProperty(globalThis.navigator, 'userAgent', {
      value: 'Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148',
      configurable: true,
    });
    expect(isMobileDevice()).toBe(true);
  });

  it('returns true for ARM + touch combination', () => {
    Object.defineProperty(globalThis.navigator, 'userAgent', {
      value: 'Mozilla/5.0 (Linux; ARM) AppleWebKit/537.36 Chrome/120.0.0.0',
      configurable: true,
    });
    Object.defineProperty(globalThis.navigator, 'maxTouchPoints', {
      value: 5,
      configurable: true,
    });
    // @ts-ignore
    globalThis.window.ontouchstart = () => {};
    expect(isMobileDevice()).toBe(true);
  });

  it('returns true for ARM + low cores', () => {
    Object.defineProperty(globalThis.navigator, 'userAgent', {
      value: 'Mozilla/5.0 (X11; Linux aarch64) AppleWebKit/537.36 Chrome/120.0.0.0',
      configurable: true,
    });
    Object.defineProperty(globalThis.navigator, 'hardwareConcurrency', {
      value: 4,
      configurable: true,
    });
    expect(isMobileDevice()).toBe(true);
  });

  it('returns true for touch + low cores (budget tablet)', () => {
    Object.defineProperty(globalThis.navigator, 'maxTouchPoints', {
      value: 10,
      configurable: true,
    });
    Object.defineProperty(globalThis.navigator, 'hardwareConcurrency', {
      value: 2,
      configurable: true,
    });
    // @ts-ignore
    globalThis.window.ontouchstart = () => {};
    expect(isMobileDevice()).toBe(true);
  });

  it('returns false for ARM desktop without touch or low cores', () => {
    // Apple Silicon Mac running as desktop — not mobile
    Object.defineProperty(globalThis.navigator, 'userAgent', {
      value: 'Mozilla/5.0 (Macintosh; Apple Silicon Mac OS X) AppleWebKit/537.36',
      configurable: true,
    });
    Object.defineProperty(globalThis.navigator, 'hardwareConcurrency', {
      value: 10,
      configurable: true,
    });
    expect(isMobileDevice()).toBe(false);
  });
});