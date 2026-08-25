"""Pass/fail reporting harness for the AngaDrive integration tests.

Every test calls :func:`check` (or the convenience wrappers :func:`check_eq`
and :func:`check_in`) to record an assertion. Each assertion prints a detailed
line describing:

  * what was being tested,
  * what was expected,
  * what actually happened (the failure detail, if any),
  * and whether it passed.

At the end of a run, :func:`summary` prints a total and the list of any failed
assertions. The process exit code is non-zero if any assertion failed.
"""

import sys

# Running totals.
PASS = 0
FAIL = 0
FAILURES = []  # names of failed assertions


def check(name, condition, detail=""):
    """Record a single assertion result.

    Args:
        name:      short human-readable description of what is being tested.
        condition: the boolean result of the assertion.
        detail:    optional explanation of the failure (what was expected vs
                   what actually happened). Shown only on failure.
    """
    global PASS, FAIL
    if condition:
        PASS += 1
        print(f"  [PASS] {name}")
    else:
        FAIL += 1
        FAILURES.append(name)
        print(f"  [FAIL] {name}")
        if detail:
            print(f"         expected: {detail}")


def check_eq(name, got, want):
    """Assert that ``got == want`` and report both values on failure."""
    check(name, got == want, f"{got!r} == {want!r}")


def check_in(name, needle, haystack):
    """Assert that ``needle`` is contained in ``haystack``."""
    check(name, needle in haystack, f"{needle!r} in {haystack!r}")


def summary():
    """Print the final totals and return True if everything passed."""
    print("\n" + "=" * 60)
    print(f"TOTAL: {PASS} passed, {FAIL} failed")
    if FAILURES:
        print("Failed assertions:")
        for f in FAILURES:
            print(f"  - {f}")
    print("=" * 60)
    return FAIL == 0


def exit_with_result(ok):
    """Exit the process with a non-zero code if any test failed."""
    sys.exit(0 if ok else 1)