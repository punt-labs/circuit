MACHINE TDDFlow
SETS
    STATE = {spec, red, green, inspecting, refactoring, done};
    TRANSITION = {writeTest, implement, inspect, finish, refactor, keepGreen}
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
                (current = green & evt = inspect & testSuitePassed = TRUE) or
                (current = inspecting & evt = finish) or
                (current = inspecting & evt = refactor) or
                (current = refactoring & evt = keepGreen & testSuitePassed = TRUE)
            )
        THEN
            IF current = spec & evt = writeTest THEN
                current := red
            ELSIF current = red & evt = implement THEN
                current := green
            ELSIF current = green & evt = inspect THEN
                current := inspecting
            ELSIF current = inspecting & evt = finish THEN
                current := done
            ELSIF current = inspecting & evt = refactor THEN
                current := refactoring
            ELSIF current = refactoring & evt = keepGreen THEN
                current := green
            END
        END
END
