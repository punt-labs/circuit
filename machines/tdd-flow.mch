MACHINE TDDFlow
SETS
    STATE = {spec, red, green, qualityReview, refactoring, done};
    TRANSITION = {writeTest, implement, reviewQuality, finish, refactor, keepGreen}
VARIABLES
    current,
    testSuitePassed,
    codeQualityPassed
INVARIANT
    current : STATE &
    testSuitePassed : BOOL &
    codeQualityPassed : BOOL
INITIALISATION
    current := spec ||
    testSuitePassed := FALSE ||
    codeQualityPassed := FALSE
OPERATIONS
    ObserveTestSuitePassed(result) =
        PRE
            result : BOOL
        THEN
            testSuitePassed := result
        END;

    ObserveCodeQualityPassed(result) =
        PRE
            result : BOOL
        THEN
            codeQualityPassed := result
        END;

    Advance(evt) =
        PRE
            evt : TRANSITION &
            current /= done &
            (
                (current = spec & evt = writeTest & not(testSuitePassed = TRUE)) or
                (current = red & evt = implement & testSuitePassed = TRUE) or
                (current = green & evt = reviewQuality & testSuitePassed = TRUE) or
                (current = qualityReview & evt = finish & codeQualityPassed = TRUE) or
                (current = qualityReview & evt = refactor & not(codeQualityPassed = TRUE)) or
                (current = refactoring & evt = keepGreen & testSuitePassed = TRUE)
            )
        THEN
            IF current = spec & evt = writeTest THEN
                current := red
            ELSIF current = red & evt = implement THEN
                current := green
            ELSIF current = green & evt = reviewQuality THEN
                current := qualityReview
            ELSIF current = qualityReview & evt = finish THEN
                current := done
            ELSIF current = qualityReview & evt = refactor THEN
                current := refactoring
            ELSIF current = refactoring & evt = keepGreen THEN
                current := green
            END
        END
END
