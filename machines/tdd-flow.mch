MACHINE TDDFlow
SETS
    STATE = {spec, red, green, refactoring, done};
    TRANSITION = {writeTest, implement, refactor, keepGreen, finish}
VARIABLES
    current,
    failingTestObserved,
    testSuitePassed
INVARIANT
    current : STATE &
    failingTestObserved : BOOL &
    testSuitePassed : BOOL
INITIALISATION
    current := spec ||
    failingTestObserved := FALSE ||
    testSuitePassed := FALSE
OPERATIONS
    ObserveFailingTestObserved(result) =
        PRE
            result : BOOL
        THEN
            failingTestObserved := result
        END;

    ObserveTestSuitePassed(result) =
        PRE
            result : BOOL
        THEN
            testSuitePassed := result
        END;

    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            (
                (current = spec & evt = writeTest & failingTestObserved = TRUE) or
                (current = red & evt = implement & testSuitePassed = TRUE) or
                (current = green & evt = refactor) or
                (current = refactoring & evt = keepGreen & testSuitePassed = TRUE) or
                (current = green & evt = finish & testSuitePassed = TRUE)
            )
        THEN
            IF current = spec & evt = writeTest THEN
                current := red
            ELSIF current = red & evt = implement THEN
                current := green
            ELSIF current = green & evt = refactor THEN
                current := refactoring
            ELSIF current = refactoring & evt = keepGreen THEN
                current := green
            ELSIF current = green & evt = finish THEN
                current := done
            END
        END
END
