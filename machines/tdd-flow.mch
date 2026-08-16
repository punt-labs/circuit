MACHINE TDDFlow
SETS
    STATE = {spec, red, green, refactoring, done};
    TRANSITION = {writeTest, implement, refactor, keepGreen, finish}
VARIABLES
    current,
    testSuitePassed
INVARIANT
    current : STATE &
    testSuitePassed : BOOL
INITIALISATION
    current := spec ||
    testSuitePassed := FALSE
OPERATIONS
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
                (current = spec & evt = writeTest & not(testSuitePassed = TRUE)) or
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
