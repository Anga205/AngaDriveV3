"""Pass/fail reporting harness for the AngaDrive integration tests.

Every test calls :func:`check` (or the convenience wrappers :func:`check_eq`
and :func:`check_in`) to record an assertion. Each assertion prints a detailed
line describing:

  * what was being tested,
  * what was expected,
  * what actually happened (the failure detail, if any),
  * whether it passed,
  * and how long the server took to respond to the request that assertion
    validates (the network round trip only, excluding test preparation and
    response verification).

Network helpers in :mod:`tests.helpers` publish the timing of the request(s)
they just made via :func:`record_steps`. A single request prints as
``(server 12.3 ms)``; a multi-request operation (e.g. an upload) prints each
call as a numbered step plus both the validated-response time and the total.

At the end of a run, :func:`summary` prints a total, the list of any failed
assertions, and the overall server-response average and slowest assertion. The
process exit code is non-zero if any assertion failed.
"""

import sys
import time
from contextlib import contextmanager

# Running totals.
PASS = 0
FAIL = 0
FAILURES = []  # names of failed assertions
CURRENT_TEST = None
TEST_RESULTS = []

# Sentinel meaning "use the timing of the most recent network operation".
_USE_LAST = object()

# Timing state.
_LAST_STEPS = []        # list of (label, seconds) from the most recent network op
_ASSERTION_TIMES = []   # list of dicts recorded for the current test
_TEST_START = None      # perf_counter() at start_test()
_SETUP_DEPTH = 0        # >0 while inside setup_scope()


def record_steps(steps):
    """Publish the timing of the network operation that just completed.

    Args:
        steps: a list of ``(label, seconds)`` pairs, one per request made by the
               operation, in order. The last step is treated as the response the
               assertion validates. Pass an empty list when no server response
               was received (e.g. a timeout) so the next assertion shows no
               server time.
    """
    global _LAST_STEPS
    _LAST_STEPS = list(steps)


@contextmanager
def setup_scope():
    """Mark assertions made inside as setup (excluded from test aggregates).

    Setup assertions still print their server timing, but they are not counted
    toward the test's average/slowest statistics because their performance is
    reported by the dedicated test that exercises the same commands.
    """
    global _SETUP_DEPTH
    _SETUP_DEPTH += 1
    try:
        yield
    finally:
        _SETUP_DEPTH -= 1


def start_test(name):
    """Start an isolated assertion scope for one integration test."""
    global PASS, FAIL, FAILURES, CURRENT_TEST
    global _LAST_STEPS, _ASSERTION_TIMES, _TEST_START, _SETUP_DEPTH
    PASS = 0
    FAIL = 0
    FAILURES = []
    CURRENT_TEST = name
    _LAST_STEPS = []
    _ASSERTION_TIMES = []
    _TEST_START = time.perf_counter()
    _SETUP_DEPTH = 0
    print(f"\n=== START {name} ===")


def _format_timing(steps):
    """Render the timing suffix for an assertion line."""
    if not steps:
        return ""
    validated_ms = steps[-1][1] * 1000.0
    if len(steps) == 1:
        return f" (server {validated_ms:.1f} ms)"
    total_ms = sum(seconds for _, seconds in steps) * 1000.0
    parts = ", ".join(
        f"step {index + 1} {label} {seconds * 1000.0:.1f} ms"
        for index, (label, seconds) in enumerate(steps)
    )
    return (f" (server {validated_ms:.1f} ms validated, "
            f"{total_ms:.1f} ms total: {parts})")


def _resolve_steps(server):
    """Turn the ``server`` argument of :func:`check` into a list of steps."""
    if server is None:
        return []
    if server is _USE_LAST:
        return list(_LAST_STEPS)
    return [("response", float(server))]


def check(name, condition, detail="", server=_USE_LAST):
    """Record a single assertion result.

    Args:
        name:      short human-readable description of what is being tested.
        condition: the boolean result of the assertion.
        detail:    optional explanation of the failure (what was expected vs
                   what actually happened). Shown only on failure.
        server:    timing source for the assertion. Defaults to the most recent
                   network operation (``_USE_LAST``). Pass ``None`` for
                   assertions that do not involve a server response (e.g. local
                   file generation), or a float number of seconds for a single
                   explicit measurement.
    """
    global PASS, FAIL
    steps = _resolve_steps(server)
    timing = _format_timing(steps)
    prefix = "[setup] " if _SETUP_DEPTH else ""

    if condition:
        PASS += 1
        print(f"  {prefix}[PASS] {name}{timing}")
    else:
        FAIL += 1
        failure = f"{CURRENT_TEST}: {name}" if CURRENT_TEST else name
        FAILURES.append(failure)
        print(f"  {prefix}[FAIL] {name}{timing}")
        if detail:
            print(f"         expected: {detail}")

    if steps and not _SETUP_DEPTH:
        _ASSERTION_TIMES.append({
            "name": name,
            "validated_ms": steps[-1][1] * 1000.0,
            "total_ms": sum(seconds for _, seconds in steps) * 1000.0,
            "steps": [(label, seconds * 1000.0) for label, seconds in steps],
        })


def check_eq(name, got, want, server=_USE_LAST):
    """Assert that ``got == want`` and report both values on failure."""
    check(name, got == want, f"{got!r} == {want!r}", server=server)


def check_in(name, needle, haystack, server=_USE_LAST):
    """Assert that ``needle`` is contained in ``haystack``."""
    check(name, needle in haystack, f"{needle!r} in {haystack!r}", server=server)


def finish_test():
    """Record and print the result of the current test."""
    duration = time.perf_counter() - _TEST_START if _TEST_START is not None else 0.0
    times = list(_ASSERTION_TIMES)

    server_avg = (sum(t["validated_ms"] for t in times) / len(times)) if times else None
    total_avg = (sum(t["total_ms"] for t in times) / len(times)) if times else None
    slowest = max(times, key=lambda t: t["validated_ms"]) if times else None

    result = {
        "name": CURRENT_TEST,
        "passed": FAIL == 0,
        "assertions_passed": PASS,
        "assertions_failed": FAIL,
        "failures": list(FAILURES),
        "duration_s": duration,
        "server_avg_ms": server_avg,
        "total_avg_ms": total_avg,
        "slowest_ms": slowest["validated_ms"] if slowest else None,
        "slowest_name": slowest["name"] if slowest else None,
        "assertion_times": times,
    }
    TEST_RESULTS.append(result)

    status = "PASS" if result["passed"] else "FAIL"
    line = (f"=== END {CURRENT_TEST}: {status} "
            f"({PASS} passed, {FAIL} failed) in {duration:.2f}s")
    if times:
        line += (f" | server avg {server_avg:.1f} ms, "
                 f"total avg {total_avg:.1f} ms, "
                 f"slowest {slowest['validated_ms']:.1f} ms ({slowest['name']})")
    line += " ==="
    print(line)
    return result


def summary(results=None):
    """Print assertion and independently identifiable test totals."""
    results = TEST_RESULTS if results is None else results
    passed_tests = sum(result["passed"] for result in results)
    failed_tests = len(results) - passed_tests
    print("\n" + "=" * 60)
    print(f"TESTS: {passed_tests} passed, {failed_tests} failed")
    print(f"ASSERTIONS: {sum(r['assertions_passed'] for r in results)} passed, "
          f"{sum(r['assertions_failed'] for r in results)} failed")

    all_times = [(result["name"], timing)
                 for result in results
                 for timing in result.get("assertion_times", [])]
    if all_times:
        server_avg = sum(t["validated_ms"] for _, t in all_times) / len(all_times)
        slow_test, slow = max(all_times, key=lambda pair: pair[1]["validated_ms"])
        print(f"SERVER: avg {server_avg:.1f} ms, "
              f"slowest {slow['validated_ms']:.1f} ms "
              f"({slow_test}: {slow['name']})")

    failed_results = [result for result in results if not result["passed"]]
    if failed_results:
        print("Failed tests:")
        for result in failed_results:
            print(f"  - {result['name']}")
            for failure in result["failures"]:
                print(f"    {failure}")
    print("=" * 60)
    return failed_tests == 0


def exit_with_result(ok):
    """Exit the process with a non-zero code if any test failed."""
    sys.exit(0 if ok else 1)