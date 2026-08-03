/**
 * Device detection utility for AngaDrive.
 *
 * Determines whether the current device is a mobile/ARM device
 * where client-side gzip compression should be avoided.
 *
 * Uses a combination of:
 *  - User-Agent string matching for known mobile/ARM patterns
 *  - Touch support detection (coarse pointer + touch events)
 *  - Hardware concurrency (low core count suggests mobile/ARM)
 *
 * Returns true if the device is likely mobile/ARM and would benefit
 * from raw (uncompressed) uploads.
 */
export function isMobileDevice(): boolean {
  // Server-side rendering guard
  if (typeof window === 'undefined' || typeof navigator === 'undefined') {
    return false;
  }

  const ua = navigator.userAgent || '';

  // Known mobile OS / device patterns
  const mobilePatterns = [
    /Android/i,
    /iPhone/i,
    /iPad/i,
    /iPod/i,
    /webOS/i,
    /BlackBerry/i,
    /Windows Phone/i,
    /Opera Mini/i,
    /IEMobile/i,
  ];

  const isMobileUA = mobilePatterns.some((p) => p.test(ua));

  // ARM-based devices (including Apple Silicon Macs running as desktop —
  // we only flag ARM if it's also mobile or has touch)
  const isARM = /\bARM\b|\barm\b|aarch64|Apple Silicon/i.test(ua);

  // Touch support: coarse pointer (touchscreen) + touch event API
  const hasTouch =
    'maxTouchPoints' in navigator &&
    navigator.maxTouchPoints > 0 &&
    'ontouchstart' in window;

  // Low core count is common on mobile/ARM devices
  const lowCores =
    'hardwareConcurrency' in navigator &&
    (navigator as Navigator & { hardwareConcurrency: number }).hardwareConcurrency <= 4;

  // Decision logic:
  // 1. Known mobile UA → mobile
  // 2. ARM + touch → mobile (e.g., ARM tablets, Apple Silicon iPad)
  // 3. ARM + low cores → mobile (e.g., Raspberry Pi, ARM Chromebook)
  // 4. Touch + low cores → mobile (e.g., budget Android tablets)
  if (isMobileUA) return true;
  if (isARM && hasTouch) return true;
  if (isARM && lowCores) return true;
  if (hasTouch && lowCores) return true;

  return false;
}