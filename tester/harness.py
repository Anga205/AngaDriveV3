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
CURRENT_TEST = None
TEST_RESULTS = []


def start_test(name):
    """Start an isolated assertion scope for one integration test."""
    global PASS, FAIL, FAILURES, CURRENT_TEST
    PASS = 0
    FAIL = 0
    FAILURES = []
    CURRENT_TEST = name
    print(f"\n=== START {name} ===")


def finish_test():
    """Record and print the result of the current test."""
    result = {
        "name": CURRENT_TEST,
        "passed": FAIL == 0,
        "assertions_passed": PASS,
        "assertions_failed": FAIL,
        "failures": list(FAILURES),
    }
    TEST_RESULTS.append(result)
    status = "PASS" if result["passed"] else "FAIL"
    print(f"=== END {CURRENT_TEST}: {status} "
          f"({PASS} passed, {FAIL} failed) ===")
    return result


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
        failure = f"{CURRENT_TEST}: {name}" if CURRENT_TEST else name
        FAILURES.append(failure)
        print(f"  [FAIL] {name}")
        if detail:
            print(f"         expected: {detail}")


def check_eq(name, got, want):
    """Assert that ``got == want`` and report both values on failure."""
    check(name, got == want, f"{got!r} == {want!r}")


def check_in(name, needle, haystack):
    """Assert that ``needle`` is contained in ``haystack``."""
    check(name, needle in haystack, f"{needle!r} in {haystack!r}")


def summary(results=None):
    """Print assertion and independently identifiable test totals."""
    results = TEST_RESULTS if results is None else results
    passed_tests = sum(result["passed"] for result in results)
    failed_tests = len(results) - passed_tests
    print("\n" + "=" * 60)
    print(f"TESTS: {passed_tests} passed, {failed_tests} failed")
    print(f"ASSERTIONS: {sum(r['assertions_passed'] for r in results)} passed, "
          f"{sum(r['assertions_failed'] for r in results)} failed")
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